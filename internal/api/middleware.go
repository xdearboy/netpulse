package api

import (
	"compress/gzip"
	"context"
	"io"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"netpulse/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var startTime = time.Now()

func SetupMiddleware(r chi.Router, rateLimit int, rateLimitWindow time.Duration) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(CORSMiddleware)
	r.Use(GzipMiddleware)
	r.Use(RequestMetricsMiddleware)
	r.Use(RateLimiter(rateLimit, rateLimitWindow))
}

var (
	totalRequests  atomic.Int64
	totalErrors    atomic.Int64
	activeRequests atomic.Int64
	requestsByPath sync.Map
	globalCache    *services.Cache
)

func SetCacheForMetrics(c *services.Cache) {
	globalCache = c
}

type pathMetrics struct {
	count        atomic.Int64
	totalDuration atomic.Int64
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func RequestMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests.Add(1)
		activeRequests.Add(1)
		defer activeRequests.Add(-1)

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		if sw.status >= 400 {
			totalErrors.Add(1)
		}

		reqID := middleware.GetReqID(r.Context())
		duration := time.Since(start)

		if sw.status >= 400 {
			log.Printf("[%s] %s %s -> %d (%v)", reqID, r.Method, r.URL.Path, sw.status, duration)
		}

		key := r.Method + " " + r.URL.Path
		if v, ok := requestsByPath.Load(key); ok {
			m := v.(*pathMetrics)
			m.count.Add(1)
			m.totalDuration.Add(duration.Nanoseconds())
		} else {
			m := &pathMetrics{}
			m.count.Add(1)
			m.totalDuration.Add(duration.Nanoseconds())
			requestsByPath.Store(key, m)
		}
	})
}

func GetMetrics() map[string]interface{} {
	paths := make(map[string]map[string]interface{})
	requestsByPath.Range(func(key, value interface{}) bool {
		m := value.(*pathMetrics)
		count := m.count.Load()
		if count > 0 {
			avg := time.Duration(m.totalDuration.Load() / count)
			paths[key.(string)] = map[string]interface{}{
				"count":    count,
				"avg_time": avg.String(),
			}
		}
		return true
	})

	result := map[string]interface{}{
		"uptime":          time.Since(startTime).String(),
		"total_requests":  totalRequests.Load(),
		"total_errors":    totalErrors.Load(),
		"active_requests": activeRequests.Load(),
		"go_routines":     runtime.NumGoroutine(),
		"paths":           paths,
	}

	if globalCache != nil {
		result["cache"] = globalCache.Stats()
	}

	return result
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

type visitor struct {
	count       int
	lastSeen    time.Time
	windowStart time.Time
}

var rl = &rateLimiter{
	visitors: make(map[string]*visitor),
	limit:    100,
	window:   time.Minute,
}

func (r *rateLimiter) allow(ip string) (bool, int, int, time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	v, exists := r.visitors[ip]

	if !exists || now.Sub(v.windowStart) > r.window {
		r.visitors[ip] = &visitor{
			count:       1,
			lastSeen:    now,
			windowStart: now,
		}
		return true, r.limit, r.limit - 1, now.Add(r.window)
	}

	v.count++
	v.lastSeen = now

	remaining := r.limit - v.count
	allowed := v.count <= r.limit

	return allowed, r.limit, max(0, remaining), v.windowStart.Add(r.window)
}

var rateLimiterCancel context.CancelFunc

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	rateLimiterCancel = cancel

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rl.mu.Lock()
				for ip, v := range rl.visitors {
					if time.Since(v.lastSeen) > rl.window*2 {
						delete(rl.visitors, ip)
					}
				}
				rl.mu.Unlock()
			}
		}
	}()
}

func StopRateLimiter() {
	if rateLimiterCancel != nil {
		rateLimiterCancel()
	}
}

// rightmost IP = closest to server = hardest to spoof
func extractClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip != "" {
				return ip
			}
		}
	}
	return r.RemoteAddr
}

func RateLimiter(limit int, window time.Duration) func(http.Handler) http.Handler {
	rl.limit = limit
	rl.window = window

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)

			allowed, limit, remaining, reset := rl.allow(ip)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))

			if !allowed {
				retryAfter := int(time.Until(reset).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded","retry_after":` + strconv.Itoa(retryAfter) + `}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, X-Request-ID, X-Cache-Hit")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// BestSpeed — latency matters more than ratio here
var gzipPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	},
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() {
			gz.Close()
			gzipPool.Put(gz)
		}()

		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	return grw.Writer.Write(b)
}
