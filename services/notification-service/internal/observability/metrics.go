package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry                *prometheus.Registry
	NotificationsMarkedRead prometheus.Counter
	NotificationRetries     prometheus.Counter
	PreferencesUpdated      prometheus.Counter
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metrics := &Metrics{
		Registry: registry,
		NotificationsMarkedRead: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notifications_marked_read_total",
			Help: "Total notifications marked read.",
		}),
		NotificationRetries: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notification_retries_total",
			Help: "Total notification retry requests.",
		}),
		PreferencesUpdated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notification_preferences_updated_total",
			Help: "Total notification preference updates.",
		}),
	}
	registry.MustRegister(metrics.NotificationsMarkedRead, metrics.NotificationRetries, metrics.PreferencesUpdated)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}
