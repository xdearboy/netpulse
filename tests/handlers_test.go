package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"netpulse/internal/api"
	"netpulse/internal/services"
	"netpulse/internal/services/sources"

	"github.com/go-chi/chi/v5"
)

type handlerMockSource struct {
	name   string
	result *sources.IPResult
	err    error
}

func (m *handlerMockSource) Name() string                                        { return m.name }
func (m *handlerMockSource) Lookup(ctx context.Context, ip string) (*sources.IPResult, error) {
	return m.result, m.err
}

func setupTestRouter(t *testing.T) *chi.Mux {
	t.Helper()
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	agg := services.NewAggregator(5 * time.Second)
	agg.AddSource(&handlerMockSource{
		name: "mock",
		result: &sources.IPResult{
			Source:       "mock",
			Country:      "US",
			CountryName:  "United States",
			City:         "Mountain View",
			Region:       "California",
			Latitude:     37.4056,
			Longitude:    -122.0775,
			ISP:          "Google LLC",
			Organization: "Google LLC",
			ASN:          15169,
			ASName:       "AS15169 Google LLC",
			Timezone:     "America/Los_Angeles",
		},
	})

	handler := api.NewHandler(nil, nil, cache, agg, 50)
	r := chi.NewRouter()
	r.Get("/api/v1/ip/{ip}", handler.GetIPInfo)
	r.Post("/api/v1/batch", handler.BatchRequest)
	return r
}

func TestGetIPInfo_ValidIP(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/ip/8.8.8.8", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result services.AggregatedResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	if result.IP != "8.8.8.8" {
		t.Errorf("IP = %s, want 8.8.8.8", result.IP)
	}
	if result.Country != "US" {
		t.Errorf("Country = %s, want US", result.Country)
	}
	if result.ASN != 15169 {
		t.Errorf("ASN = %d, want 15169", result.ASN)
	}
	if result.Organization != "Google LLC" {
		t.Errorf("Organization = %s, want Google LLC", result.Organization)
	}
	if len(result.SourcesUsed) == 0 {
		t.Error("expected at least 1 source used")
	}
}

func TestGetIPInfo_InvalidIP(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/ip/not-an-ip", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}

	var errResp api.ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error != "Invalid IP address" {
		t.Errorf("error = %s", errResp.Error)
	}
}

func TestGetIPInfo_CacheHit(t *testing.T) {
	r := setupTestRouter(t)

	req1 := httptest.NewRequest("GET", "/api/v1/ip/1.1.1.1", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: status = %d", w1.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/ip/1.1.1.1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("second request: status = %d", w2.Code)
	}

	var r1, r2 services.AggregatedResult
	json.NewDecoder(w1.Body).Decode(&r1)
	json.NewDecoder(w2.Body).Decode(&r2)

	if r1.Country != r2.Country {
		t.Errorf("cached result differs: %s vs %s", r1.Country, r2.Country)
	}
}

func TestGetIPInfo_IPv6(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/ip/2001:4860:4860::8888", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result services.AggregatedResult
	json.NewDecoder(w.Body).Decode(&result)

	if result.Type != "IPv6" {
		t.Errorf("type = %s, want IPv6", result.Type)
	}
}

func TestBatchRequest(t *testing.T) {
	r := setupTestRouter(t)

	batch := api.BatchRequest{
		IPs: []string{"8.8.8.8", "1.1.1.1"},
	}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest("POST", "/api/v1/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp api.BatchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.IPs) != 2 {
		t.Errorf("expected 2 IPs in response, got %d", len(resp.IPs))
	}

	if _, ok := resp.IPs["8.8.8.8"]; !ok {
		t.Error("missing 8.8.8.8 in response")
	}
	if _, ok := resp.IPs["1.1.1.1"]; !ok {
		t.Error("missing 1.1.1.1 in response")
	}
}

func TestBatchRequest_InvalidBody(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest("POST", "/api/v1/batch", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBatchRequest_EmptyBatch(t *testing.T) {
	r := setupTestRouter(t)

	batch := api.BatchRequest{}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest("POST", "/api/v1/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestBatchRequest_ExceedsLimit(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	agg := services.NewAggregator(5 * time.Second)
	handler := api.NewHandler(nil, nil, cache, agg, 2)
	r := chi.NewRouter()
	r.Post("/api/v1/batch", handler.BatchRequest)

	batch := api.BatchRequest{
		IPs: []string{"8.8.8.8", "1.1.1.1", "9.9.9.9"},
	}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest("POST", "/api/v1/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetIPInfo_CacheHitHeader(t *testing.T) {
	r := setupTestRouter(t)

	req1 := httptest.NewRequest("GET", "/api/v1/ip/1.1.1.1", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	cacheHit1 := w1.Header().Get("X-Cache-Hit")
	if cacheHit1 != "false" {
		t.Errorf("first request cache hit = %s, want false", cacheHit1)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/ip/1.1.1.1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	cacheHit2 := w2.Header().Get("X-Cache-Hit")
	if cacheHit2 != "true" {
		t.Errorf("second request cache hit = %s, want true", cacheHit2)
	}
}

func TestHealthCheck(t *testing.T) {
	cache, err := services.NewCache(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	agg := services.NewAggregator(5 * time.Second)
	agg.AddSource(&handlerMockSource{
		name:   "mock",
		result: &sources.IPResult{Country: "US"},
	})
	handler := api.NewHandler(nil, nil, cache, agg, 50)
	r := chi.NewRouter()
	r.Get("/health", handler.HealthCheck)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp api.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Status != "ok" {
		t.Errorf("status = %s, want ok", resp.Status)
	}
	if resp.Summary.TotalSources != 1 {
		t.Errorf("total_sources = %d, want 1", resp.Summary.TotalSources)
	}
	if resp.Summary.HealthySources != 1 {
		t.Errorf("healthy_sources = %d, want 1", resp.Summary.HealthySources)
	}
}

func TestBatchRequest_BodyTooLarge(t *testing.T) {
	r := setupTestRouter(t)

	ips := make([]string, 200000)
	for i := range ips {
		ips[i] = "1.1.1.1"
	}
	body, _ := json.Marshal(api.BatchRequest{IPs: ips})

	req := httptest.NewRequest("POST", "/api/v1/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for oversized body", w.Code)
	}
}
