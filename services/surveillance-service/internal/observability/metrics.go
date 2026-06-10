package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry               *prometheus.Registry
	AlertsCreated          prometheus.Counter
	AlertsAcknowledged     prometheus.Counter
	AlertsResolved         prometheus.Counter
	AlertsDismissed        prometheus.Counter
	RuleMatches            prometheus.CounterVec
	RuleExecutions         prometheus.CounterVec
	KafkaMessages          prometheus.CounterVec
	KafkaPublishErrors     prometheus.Counter
	RuleDuration           prometheus.Histogram
	EventsRetried          prometheus.CounterVec
	EventsDeadlettered     prometheus.CounterVec
	ProcessingAttempts     prometheus.CounterVec
	ProcessingDuration     prometheus.HistogramVec
	DuplicateSkipped       prometheus.CounterVec
	RuleConfigUpdates      prometheus.CounterVec
	RuleConfigReloads      prometheus.CounterVec
	RuleDisabledSkips      prometheus.CounterVec
	RuleConfigCacheEntries prometheus.Gauge
	RuleSimulationRequests prometheus.CounterVec
	RuleSimulationDuration prometheus.HistogramVec
	RuleSimulationMatches  prometheus.CounterVec
	RuleSimulationFailures prometheus.CounterVec
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
		EventsRetried: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_events_retried_total",
			Help: "Total surveillance source events retried.",
		}, []string{"topic"}),
		EventsDeadlettered: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_events_deadlettered_total",
			Help: "Total surveillance source events published to DLQ.",
		}, []string{"topic"}),
		ProcessingAttempts: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_event_processing_attempts_total",
			Help: "Total surveillance event processing attempts by status.",
		}, []string{"topic", "status"}),
		ProcessingDuration: *prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "surveillance_event_processing_duration_seconds",
			Help:    "Surveillance source event processing duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}, []string{"topic"}),
		DuplicateSkipped: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_duplicate_events_skipped_total",
			Help: "Total duplicate surveillance alerts skipped.",
		}, []string{"topic"}),
		RuleConfigUpdates: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_config_updates_total",
			Help: "Total surveillance rule config updates.",
		}, []string{"rule_name", "action"}),
		RuleConfigReloads: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_config_reload_total",
			Help: "Total surveillance rule config cache reloads.",
		}, []string{"status"}),
		RuleDisabledSkips: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_disabled_skips_total",
			Help: "Total surveillance rule evaluations skipped because the rule is disabled.",
		}, []string{"rule_name"}),
		RuleConfigCacheEntries: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "surveillance_rule_config_cache_entries",
			Help: "Current number of surveillance rule config entries cached.",
		}),
		RuleSimulationRequests: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_simulation_requests_total",
			Help: "Total surveillance rule simulation requests.",
		}, []string{"rule_name", "status"}),
		RuleSimulationDuration: *prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "surveillance_rule_simulation_duration_seconds",
			Help:    "Surveillance rule simulation duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}, []string{"rule_name", "status"}),
		RuleSimulationMatches: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_simulation_matches_total",
			Help: "Total matched events found by surveillance rule dry-run simulations.",
		}, []string{"rule_name"}),
		RuleSimulationFailures: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "surveillance_rule_simulation_failures_total",
			Help: "Total failed surveillance rule simulations.",
		}, []string{"rule_name"}),
	}
	registry.MustRegister(metrics.AlertsCreated, metrics.AlertsAcknowledged, metrics.AlertsResolved, metrics.AlertsDismissed, &metrics.RuleMatches, &metrics.RuleExecutions, &metrics.KafkaMessages, metrics.KafkaPublishErrors, metrics.RuleDuration, &metrics.EventsRetried, &metrics.EventsDeadlettered, &metrics.ProcessingAttempts, &metrics.ProcessingDuration, &metrics.DuplicateSkipped, &metrics.RuleConfigUpdates, &metrics.RuleConfigReloads, &metrics.RuleDisabledSkips, metrics.RuleConfigCacheEntries, &metrics.RuleSimulationRequests, &metrics.RuleSimulationDuration, &metrics.RuleSimulationMatches, &metrics.RuleSimulationFailures)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveRule(start time.Time) {
	m.RuleDuration.Observe(time.Since(start).Seconds())
}
