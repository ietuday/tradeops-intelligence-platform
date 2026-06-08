package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry           *prometheus.Registry
	AlertsCreated      prometheus.Counter
	AlertsAcknowledged prometheus.Counter
	AlertsResolved     prometheus.Counter
	AlertsDismissed    prometheus.Counter
	RuleMatches        prometheus.CounterVec
	RuleExecutions     prometheus.CounterVec
	KafkaMessages      prometheus.CounterVec
	KafkaPublishErrors prometheus.Counter
	RuleDuration       prometheus.Histogram
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metrics := &Metrics{
		Registry: registry,
		AlertsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "surveillance_alerts_created_total",
			Help: "Total surveillance alerts created.",
		}),
		AlertsAcknowledged: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "surveillance_alerts_acknowledged_total",
			Help: "Total surveillance alerts acknowledged.",
		}),
		AlertsResolved: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "surveillance_alerts_resolved_total",
			Help: "Total surveillance alerts resolved.",
		}),
		AlertsDismissed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "surveillance_alerts_dismissed_total",
			Help: "Total surveillance alerts dismissed.",
		}),
		RuleMatches: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_matches_total",
			Help: "Total surveillance rule matches.",
		}, []string{"rule"}),
		RuleExecutions: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_executions_total",
			Help: "Total surveillance rule executions.",
		}, []string{"rule", "topic"}),
		KafkaMessages: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_kafka_messages_total",
			Help: "Total Kafka messages consumed by surveillance.",
		}, []string{"topic"}),
		KafkaPublishErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "surveillance_kafka_publish_errors_total",
			Help: "Total surveillance Kafka publish errors.",
		}),
		RuleDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "surveillance_rule_duration_seconds",
			Help:    "Surveillance rule execution duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),
	}
	registry.MustRegister(metrics.AlertsCreated, metrics.AlertsAcknowledged, metrics.AlertsResolved, metrics.AlertsDismissed, &metrics.RuleMatches, &metrics.RuleExecutions, &metrics.KafkaMessages, metrics.KafkaPublishErrors, metrics.RuleDuration)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveRule(start time.Time) {
	m.RuleDuration.Observe(time.Since(start).Seconds())
}
