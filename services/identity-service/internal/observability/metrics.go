package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry        *prometheus.Registry
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tradeops_identity_service_http_requests_total",
		Help: "Total HTTP requests received by the Identity Service.",
	}, []string{"method", "route", "status_code"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tradeops_identity_service_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds for the Identity Service.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	}, []string{"method", "route", "status_code"})
	registry.MustRegister(requests, duration)
	return &Metrics{Registry: registry, RequestsTotal: requests, RequestDuration: duration}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(recorder, r)
		route := routePattern(r)
		labels := prometheus.Labels{"method": r.Method, "route": route, "status_code": strconv.Itoa(recorder.status)}
		m.RequestsTotal.With(labels).Inc()
		m.RequestDuration.With(labels).Observe(time.Since(start).Seconds())
	})
}

func routePattern(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if r.URL != nil && r.URL.Path != "" {
		return r.URL.Path
	}
	return "unknown"
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
