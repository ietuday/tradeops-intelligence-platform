package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRoutePatternFallbacks(t *testing.T) {
	if got := routePattern(nil); got != "unknown" {
		t.Fatalf("routePattern(nil) = %q, want unknown", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	if got := routePattern(req); got != "/health" {
		t.Fatalf("routePattern without chi context = %q, want /health", got)
	}

	req = &http.Request{}
	if got := routePattern(req); got != "unknown" {
		t.Fatalf("routePattern without URL = %q, want unknown", got)
	}
}

func TestRoutePatternWithChiContext(t *testing.T) {
	router := chi.NewRouter()
	var got string
	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		got = routePattern(r)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/123", nil))

	if got != "/users/{id}" {
		t.Fatalf("routePattern with chi context = %q, want /users/{id}", got)
	}
}

func TestMetricsMiddlewareDoesNotPanicWithoutChiRouteContext(t *testing.T) {
	metrics := NewMetrics()
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
}
