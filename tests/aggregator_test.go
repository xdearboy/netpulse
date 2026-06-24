package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"netpulse/internal/services"
	"netpulse/internal/services/sources"
)

type mockSource struct {
	name   string
	result *sources.IPResult
	err    error
}

func (m *mockSource) Name() string { return m.name }
func (m *mockSource) Lookup(ctx context.Context, ip string) (*sources.IPResult, error) {
	return m.result, m.err
}

func newMockResult(country, city, org string, asn int) *sources.IPResult {
	return &sources.IPResult{
		Source:       "mock",
		Country:      country,
		City:         city,
		Organization: org,
		ASN:          asn,
		Latitude:     55.75,
		Longitude:    37.62,
		Timezone:     "Europe/Moscow",
		ISP:          "TestISP",
	}
}

func TestAggregator_ConsensusCountry(t *testing.T) {
	agg := services.NewAggregator(5 * time.Second)
	agg.AddSource(&mockSource{"src1", newMockResult("RU", "Moscow", "Rostelecom", 12389), nil})
	agg.AddSource(&mockSource{"src2", newMockResult("RU", "Moscow", "Rostelecom", 12389), nil})
	agg.AddSource(&mockSource{"src3", newMockResult("US", "New York", "Google", 15169), nil})

	result := agg.Lookup(context.Background(), "8.8.8.8")

	if result.Country != "RU" {
		t.Errorf("expected consensus country RU, got %s", result.Country)
	}
	if result.City != "Moscow" {
		t.Errorf("expected consensus city Moscow, got %s", result.City)
	}
	if result.ASN != 12389 {
		t.Errorf("expected consensus ASN 12389, got %d", result.ASN)
	}
}

func TestAggregator_AllSourcesFailed(t *testing.T) {
	agg := services.NewAggregator(2 * time.Second)
	agg.AddSource(&mockSource{"src1", nil, fmt.Errorf("timeout")})
	agg.AddSource(&mockSource{"src2", nil, fmt.Errorf("rate limit")})

	result := agg.Lookup(context.Background(), "1.2.3.4")

	if result.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %s", result.IP)
	}
	if len(result.SourcesUsed) != 0 {
		t.Errorf("expected 0 sources used, got %d", len(result.SourcesUsed))
	}
	if len(result.SourcesFailed) != 2 {
		t.Errorf("expected 2 sources failed, got %d", len(result.SourcesFailed))
	}
}

func TestAggregator_MixedResults(t *testing.T) {
	agg := services.NewAggregator(5 * time.Second)
	agg.AddSource(&mockSource{"src1", newMockResult("RU", "Moscow", "Rostelecom", 12389), nil})
	agg.AddSource(&mockSource{"src2", nil, fmt.Errorf("error")})
	agg.AddSource(&mockSource{"src3", newMockResult("RU", "Moscow", "Rostelecom", 12389), nil})

	result := agg.Lookup(context.Background(), "10.0.0.1")

	if len(result.SourcesUsed) != 2 {
		t.Errorf("expected 2 sources used, got %d", len(result.SourcesUsed))
	}
	if len(result.SourcesFailed) != 1 {
		t.Errorf("expected 1 source failed, got %d", len(result.SourcesFailed))
	}
	if result.Country != "RU" {
		t.Errorf("expected RU, got %s", result.Country)
	}
}

func TestAggregator_MedianCoordinates(t *testing.T) {
	agg := services.NewAggregator(5 * time.Second)

	r1 := &sources.IPResult{Latitude: 55.0, Longitude: 37.0}
	r2 := &sources.IPResult{Latitude: 56.0, Longitude: 38.0}
	r3 := &sources.IPResult{Latitude: 100.0, Longitude: 200.0}

	agg.AddSource(&mockSource{"src1", r1, nil})
	agg.AddSource(&mockSource{"src2", r2, nil})
	agg.AddSource(&mockSource{"src3", r3, nil})

	result := agg.Lookup(context.Background(), "1.1.1.1")

	if result.Latitude != 56.0 {
		t.Errorf("expected median lat 56, got %f", result.Latitude)
	}
	if result.Longitude != 38.0 {
		t.Errorf("expected median lon 38, got %f", result.Longitude)
	}
}

func TestAggregator_EmptySources(t *testing.T) {
	agg := services.NewAggregator(2 * time.Second)
	result := agg.Lookup(context.Background(), "1.1.1.1")

	if result.IP != "1.1.1.1" {
		t.Errorf("expected IP 1.1.1.1, got %s", result.IP)
	}
	if result.Country != "" {
		t.Errorf("expected empty country, got %s", result.Country)
	}
}

func TestAggregator_IPv6(t *testing.T) {
	agg := services.NewAggregator(5 * time.Second)
	agg.AddSource(&mockSource{"src1", newMockResult("US", "San Francisco", "Cloudflare", 13335), nil})

	result := agg.Lookup(context.Background(), "2606:4700:4700::1111")

	if result.Type != "IPv6" {
		t.Errorf("expected type IPv6, got %s", result.Type)
	}
	if result.Country != "US" {
		t.Errorf("expected US, got %s", result.Country)
	}
}

func TestAggregator_ContextCancellation(t *testing.T) {
	agg := services.NewAggregator(5 * time.Second)
	agg.AddSource(&mockSource{"src1", newMockResult("US", "NYC", "Test", 12345), nil})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := agg.Lookup(ctx, "1.1.1.1")

	if result.IP != "1.1.1.1" {
		t.Errorf("expected IP 1.1.1.1, got %s", result.IP)
	}
}
