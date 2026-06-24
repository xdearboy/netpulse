package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	AggregatorTimeout   time.Duration
	CacheTTL            time.Duration
	RateLimit           int
	RateLimitWindow     time.Duration
	BatchMaxSize        int
	IPGeolocationAPIKey string
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		AggregatorTimeout:   getDurationEnv("AGGREGATOR_TIMEOUT", 10*time.Second),
		CacheTTL:            getDurationEnv("CACHE_TTL", 10*time.Minute),
		RateLimit:           getIntEnv("RATE_LIMIT", 100),
		RateLimitWindow:     getDurationEnv("RATE_LIMIT_WINDOW", time.Minute),
		BatchMaxSize:        getIntEnv("BATCH_MAX_SIZE", 50),
		IPGeolocationAPIKey: getEnv("IPGEOLOCATION_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
