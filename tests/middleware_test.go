package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"netpulse/internal/api"
	"netpulse/internal/services"
	"netpulse/internal/services/sources"

	"github.com/go-chi/chi/v5"
)

func TestRateLimiter_AllowsRequests(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	agg := services.NewAggregator(5 * time.Second)
	handler := api.NewHandler(nil, nil, cache, agg, 50)

	r := chi.NewRouter()
	api.SetupMiddleware(r, 5, time.Minute)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want 200", w.Code)
	}

	limit := w.Header().Get("X-RateLimit-Limit")
	if limit != "5" {
		t.Errorf("X-RateLimit-Limit = %s, want 5", limit)
	}
	remaining := w.Header().Get("X-RateLimit-Remaining")
	if remaining != "4" {
		t.Errorf("X-RateLimit-Remaining = %s, want 4", remaining)
	}
	_ = handler
}

func TestRateLimiter_BlocksAfterLimit(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	agg := services.NewAggregator(5 * time.Second)
	handler := api.NewHandler(nil, nil, cache, agg, 50)

	r := chi.NewRouter()
	api.SetupMiddleware(r, 2, time.Minute)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request: status = %d, want 429", w.Code)
	}

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("expected Retry-After header")
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "rate limit exceeded" {
		t.Errorf("error = %v, want 'rate limit exceeded'", body["error"])
	}
	_ = handler
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	agg := services.NewAggregator(5 * time.Second)
	handler := api.NewHandler(nil, nil, cache, agg, 50)

	r := chi.NewRouter()
	api.SetupMiddleware(r, 1, time.Minute)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "2.2.2.2:1234"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w1.Code != http.StatusOK {
		t.Errorf("IP1: status = %d, want 200", w1.Code)
	}
	if w2.Code != http.StatusOK {
		t.Errorf("IP2: status = %d, want 200", w2.Code)
	}
	_ = handler
}

func TestRateLimiter_XForwardedFor(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	agg := services.NewAggregator(5 * time.Second)
	handler := api.NewHandler(nil, nil, cache, agg, 50)

	r := chi.NewRouter()
	api.SetupMiddleware(r, 1, time.Minute)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 70.41.3.18")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want 200", w.Code)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	req2.Header.Set("X-Forwarded-For", "203.0.113.1, 70.41.3.18")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request same IP: status = %d, want 429", w2.Code)
	}
	_ = handler
}

func TestCache_Stats(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	type testVal struct {
		Name string `json:"name"`
	}

	for i := 0; i < 5; i++ {
		cache.Set("key"+string(rune('a'+i)), testVal{Name: "test"})
	}

	for i := 0; i < 3; i++ {
		var v testVal
		cache.Get("key"+string(rune('a'+i)), &v)
	}

	for i := 0; i < 3; i++ {
		var v testVal
		cache.Get("missing"+string(rune('a'+i)), &v)
	}

	stats := cache.Stats()
	if stats.Hits != 3 {
		t.Errorf("hits = %d, want 3", stats.Hits)
	}
	if stats.Misses != 3 {
		t.Errorf("misses = %d, want 3", stats.Misses)
	}
	if stats.HitRatio != 0.5 {
		t.Errorf("hit_ratio = %f, want 0.5", stats.HitRatio)
	}
}

func TestHealthCheck_Degraded(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	agg := services.NewAggregator(5 * time.Second)
	agg.AddSource(&handlerMockSource{
		name: "good",
		result: &sources.IPResult{Country: "US"},
	})
	agg.AddSource(&handlerMockSource{
		name: "bad",
		err:  fmt.Errorf("connection refused"),
	})
	handler := api.NewHandler(nil, nil, cache, agg, 50)
	r := chi.NewRouter()
	r.Get("/health", handler.HealthCheck)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusPartialContent {
		t.Errorf("status = %d, want 206", w.Code)
	}

	var resp api.HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Status != "degraded" {
		t.Errorf("status = %s, want degraded", resp.Status)
	}
	if resp.Summary.TotalSources != 2 {
		t.Errorf("total_sources = %d, want 2", resp.Summary.TotalSources)
	}
	if resp.Summary.HealthySources != 1 {
		t.Errorf("healthy_sources = %d, want 1", resp.Summary.HealthySources)
	}
}

func TestMetrics(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	api.SetCacheForMetrics(cache)
	agg := services.NewAggregator(5 * time.Second)
	handler := api.NewHandler(nil, nil, cache, agg, 50)
	r := chi.NewRouter()
	r.Get("/metrics", handler.Metrics)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var metrics map[string]interface{}
	json.NewDecoder(w.Body).Decode(&metrics)

	if _, ok := metrics["uptime"]; !ok {
		t.Error("missing uptime in metrics")
	}
	if _, ok := metrics["total_requests"]; !ok {
		t.Error("missing total_requests in metrics")
	}
	if _, ok := metrics["cache"]; !ok {
		t.Error("missing cache stats in metrics")
	}
}
