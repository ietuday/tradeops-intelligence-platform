package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry           *prometheus.Registry
	EventsProcessed    prometheus.CounterVec
	EventsFailed       prometheus.CounterVec
	LogsCreated        prometheus.CounterVec
	DuplicatesSkipped  prometheus.CounterVec
	EventsDeadlettered prometheus.CounterVec
	EventsRetried      prometheus.CounterVec
	ProcessingAttempts prometheus.CounterVec
	ProcessingDuration prometheus.HistogramVec
	ExportRequests     prometheus.CounterVec
	KafkaPublishErrors prometheus.Counter
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metrics := &Metrics{
		Registry: registry,
		EventsProcessed: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_processed_total",
			Help: "Total audit source events processed.",
		}, []string{"topic"}),
		EventsFailed: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_failed_total",
			Help: "Total audit source event processing failures.",
		}, []string{"topic"}),
		LogsCreated: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_logs_created_total",
			Help: "Total audit logs created.",
		}, []string{"service_name", "event_type", "severity"}),
		DuplicatesSkipped: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_logs_duplicate_skipped_total",
			Help: "Total duplicate audit logs skipped.",
		}, []string{"topic"}),
		EventsDeadlettered: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_deadlettered_total",
			Help: "Total audit source events published to DLQ.",
		}, []string{"topic"}),
		EventsRetried: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_retried_total",
			Help: "Total audit source events retried.",
		}, []string{"topic"}),
		ProcessingAttempts: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_event_processing_attempts_total",
			Help: "Total audit event processing attempts by status.",
		}, []string{"topic", "status"}),
		ProcessingDuration: *prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "audit_event_processing_duration_seconds",
			Help:    "Audit source event processing duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}, []string{"topic"}),
		ExportRequests: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_export_requests_total",
			Help: "Total audit export requests.",
		}, []string{"format"}),
		KafkaPublishErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "audit_kafka_publish_errors_total",
			Help: "Total audit Kafka publish errors.",
		}),
	}
	registry.MustRegister(&metrics.EventsProcessed, &metrics.EventsFailed, &metrics.LogsCreated, &metrics.DuplicatesSkipped, &metrics.EventsDeadlettered, &metrics.EventsRetried, &metrics.ProcessingAttempts, &metrics.ProcessingDuration, &metrics.ExportRequests, metrics.KafkaPublishErrors)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveProcessing(topic string, start time.Time) {
	m.ProcessingDuration.WithLabelValues(topic).Observe(time.Since(start).Seconds())
}
