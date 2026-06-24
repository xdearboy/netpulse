package tests

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"netpulse/internal/api"
	"netpulse/internal/services"
	"netpulse/internal/services/sources"
)

func testStaticFS(t *testing.T) fs.FS {
	t.Helper()
	return fstest.MapFS{
		"index.html": {Data: []byte("<html>netpulse</html>")},
		"docs.html":  {Data: []byte("<html>scalar docs</html>")},
	}
}

func setupFullRouter(t *testing.T) http.Handler {
	t.Helper()
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
	return api.NewRouter(handler, 100, time.Minute, testStaticFS(t))
}

func TestIntegration_RootLanding(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET / status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %s, want text/html", ct)
	}
}

func TestIntegration_DocsServesScalar(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/docs", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /docs status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %s, want text/html", ct)
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("empty response body")
	}
}

func TestIntegration_OpenAPIJSON(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /openapi.json status = %d, want 200", w.Code)
	}
}

func TestIntegration_IPGeolocation(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/api/v1/ip/8.8.8.8", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/ip/8.8.8.8 status = %d, want 200", w.Code)
	}
}

func TestIntegration_HealthCheck(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /health status = %d, want 200 or 206", w.Code)
	}
}

func TestIntegration_Metrics(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /metrics status = %d, want 200", w.Code)
	}
}

func TestIntegration_InvalidIP(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/api/v1/ip/not-an-ip", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GET /api/v1/ip/not-an-ip status = %d, want 400", w.Code)
	}
}

func TestIntegration_404(t *testing.T) {
	r := setupFullRouter(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected non-200 for nonexistent route")
	}
}
