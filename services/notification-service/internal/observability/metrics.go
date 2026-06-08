package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry                *prometheus.Registry
	NotificationsMarkedRead prometheus.Counter
	NotificationRetries     prometheus.Counter
	PreferencesUpdated      prometheus.Counter
	EventsProcessed         prometheus.Counter
	EventsFailed            prometheus.Counter
	NotificationsCreated    prometheus.Counter
	DeliveryAttempts        prometheus.Counter
	DeliveryFailures        prometheus.Counter
	StatusUpdates           prometheus.Counter
	DeliveryDuration        prometheus.Histogram
	EventsRetried           prometheus.CounterVec
	EventsDeadlettered      prometheus.CounterVec
	ProcessingAttempts      prometheus.CounterVec
	ProcessingDuration      prometheus.HistogramVec
	DuplicateSkipped        prometheus.CounterVec
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
		EventsProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notification_events_processed_total",
			Help: "Total notification source events processed.",
		}),
		EventsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notification_events_failed_total",
			Help: "Total notification source event processing failures.",
		}),
		NotificationsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notifications_created_total",
			Help: "Total notifications created.",
		}),
		DeliveryAttempts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notification_delivery_attempts_total",
			Help: "Total notification delivery attempts.",
		}),
		DeliveryFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notification_delivery_failures_total",
			Help: "Total notification delivery failures.",
		}),
		StatusUpdates: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "notification_status_updates_total",
			Help: "Total notification status updates.",
		}),
		DeliveryDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "notification_delivery_duration_seconds",
			Help:    "Notification delivery duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}),
		EventsRetried: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notification_events_retried_total",
			Help: "Total notification source events retried.",
		}, []string{"topic"}),
		EventsDeadlettered: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notification_events_deadlettered_total",
			Help: "Total notification source events published to DLQ.",
		}, []string{"topic"}),
		ProcessingAttempts: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notification_event_processing_attempts_total",
			Help: "Total notification event processing attempts by status.",
		}, []string{"topic", "status"}),
		ProcessingDuration: *prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "notification_event_processing_duration_seconds",
			Help:    "Notification source event processing duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}, []string{"topic"}),
		DuplicateSkipped: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notification_duplicate_events_skipped_total",
			Help: "Total duplicate notifications skipped.",
		}, []string{"topic"}),
	}
	registry.MustRegister(metrics.NotificationsMarkedRead, metrics.NotificationRetries, metrics.PreferencesUpdated, metrics.EventsProcessed, metrics.EventsFailed, metrics.NotificationsCreated, metrics.DeliveryAttempts, metrics.DeliveryFailures, metrics.StatusUpdates, metrics.DeliveryDuration, &metrics.EventsRetried, &metrics.EventsDeadlettered, &metrics.ProcessingAttempts, &metrics.ProcessingDuration, &metrics.DuplicateSkipped)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveDelivery(start time.Time) {
	m.DeliveryDuration.Observe(time.Since(start).Seconds())
}
