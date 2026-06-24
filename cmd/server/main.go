package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netpulse/internal/api"
	"netpulse/internal/config"
	"netpulse/internal/services"
	"netpulse/internal/services/sources"
)

//go:embed static
var staticFiles embed.FS

func main() {
	cfg := config.Load()

	cache, err := services.NewCache(cfg.CacheTTL)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
	defer cache.Close()
	api.SetCacheForMetrics(cache)

	agg := services.NewAggregator(cfg.AggregatorTimeout)
	agg.AddSource(sources.NewIPAPIClient())
	agg.AddSource(sources.NewIPWhoisClient())
	agg.AddSource(sources.NewIPInfoSourceClient(cfg.IPInfoToken))
	agg.AddSource(sources.NewDBIPClient())
	if cfg.IPGeolocationAPIKey != "" {
		agg.AddSource(sources.NewIPGeolocationClient(cfg.IPGeolocationAPIKey))
	}
	log.Printf("Aggregator initialized with %d sources", agg.SourceCount())

	ripeClient := services.NewRipeClient()
	ipinfoClient := services.NewIPInfoClient(cfg.IPInfoToken)
	handler := api.NewHandler(ripeClient, ipinfoClient, cache, agg, cfg.BatchMaxSize)

	staticFS, _ := fs.Sub(staticFiles, "static")
	r := api.NewRouter(handler, cfg.RateLimit, cfg.RateLimitWindow, staticFS)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s", server.Addr)
		log.Printf("Swagger UI: http://localhost:%s/docs", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	api.StopRateLimiter()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
