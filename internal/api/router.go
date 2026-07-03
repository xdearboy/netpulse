package api

import (
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler, rateLimit int, rateLimitWindow time.Duration, staticFS fs.FS) chi.Router {
	r := chi.NewRouter()
	SetupMiddleware(r, rateLimit, rateLimitWindow)

	docsHTML, _ := fs.ReadFile(staticFS, "docs.html")
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/docs" && !strings.Contains(r.Header.Get("Accept"), "application/json") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(docsHTML)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	SetupAPI(r, handler, rateLimit, rateLimitWindow)

	fileServer := http.FileServer(http.FS(staticFS))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" || r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	return r
}
