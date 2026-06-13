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
	router.Get("/notifications/{id}", func(w http.ResponseWriter, r *http.Request) {
		got = routePattern(r)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/notifications/123", nil))

	if got != "/notifications/{id}" {
		t.Fatalf("routePattern with chi context = %q, want /notifications/{id}", got)
	}
}

func TestTracingHandlersDoNotPanicWithoutChiRouteContext(t *testing.T) {
	handler := HTTPHandler("notification-service", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	traced := TraceAttributes("notification-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	traced.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ready", nil))
}
