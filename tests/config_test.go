package tests

import (
	"os"
	"testing"
	"time"

	"netpulse/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	cfg := config.Load()
	if cfg.Port != "8080" {
		t.Errorf("default port = %s, want 8080", cfg.Port)
	}
	if cfg.AggregatorTimeout != 10*time.Second {
		t.Errorf("default timeout = %v, want 10s", cfg.AggregatorTimeout)
	}
	if cfg.CacheTTL != 10*time.Minute {
		t.Errorf("default cache TTL = %v, want 10m", cfg.CacheTTL)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("AGGREGATOR_TIMEOUT", "15")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("AGGREGATOR_TIMEOUT")

	cfg := config.Load()
	if cfg.Port != "9090" {
		t.Errorf("port = %s, want 9090", cfg.Port)
	}
	if cfg.AggregatorTimeout != 15*time.Second {
		t.Errorf("timeout = %v, want 15s", cfg.AggregatorTimeout)
	}
}
