package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry                        *prometheus.Registry
	OrdersCreated                   prometheus.Counter
	OrdersAccepted                  prometheus.Counter
	OrdersFilled                    prometheus.Counter
	OrdersRejected                  prometheus.Counter
	OrdersCancelled                 prometheus.Counter
	OrdersAmended                   prometheus.Counter
	OrdersPartial                   prometheus.Counter
	TradesExecuted                  prometheus.Counter
	IdempotencyReplays              prometheus.Counter
	KafkaPublishErrors              prometheus.Counter
	ProcessingDuration              prometheus.Histogram
	OutboxClaimed                   prometheus.Counter
	OutboxPublished                 *prometheus.CounterVec
	OutboxPublishErrors             *prometheus.CounterVec
	OutboxTerminalFailures          *prometheus.CounterVec
	OutboxPending                   prometheus.Gauge
	OutboxProcessing                prometheus.Gauge
	OutboxFailed                    prometheus.Gauge
	OutboxOldestPendingAge          prometheus.Gauge
	OutboxPublishDuration           *prometheus.HistogramVec
	OutboxBatchSize                 prometheus.Histogram
	OutboxClaimDuration             prometheus.Histogram
	OutboxRetryDelay                prometheus.Histogram
	OutboxLeaseRecoveries           prometheus.Counter
	OrderExpiryPolls                *prometheus.CounterVec
	OrderExpiryDueOrders            prometheus.Gauge
	OrderExpiryClaimed              prometheus.Counter
	OrdersExpired                   *prometheus.CounterVec
	OrderExpiryErrors               *prometheus.CounterVec
	OrderExpiryConflicts            prometheus.Counter
	OrderExpiryDuration             *prometheus.HistogramVec
	OrderExpiryBatchSize            prometheus.Histogram
	OrderExpiryLag                  prometheus.Histogram
	OrderExpiryOldestDueAge         prometheus.Gauge
	OrderExpiryReconciliationErrors prometheus.Counter
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metrics := &Metrics{
		Registry: registry,
		OrdersCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Total orders created.",
		}),
		OrdersAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "orders_accepted_total",
			Help: "Total orders accepted.",
		}),
		OrdersFilled: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "orders_filled_total",
			Help: "Total orders filled.",
		}),
		OrdersRejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "orders_rejected_total",
			Help: "Total orders rejected.",
		}),
		OrdersCancelled: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "orders_cancelled_total",
			Help: "Total orders cancelled.",
		}),
		OrdersAmended: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_orders_amended_total",
			Help: "Total orders amended.",
		}),
		OrdersPartial: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_orders_partially_filled_total",
			Help: "Total partially filled orders.",
		}),
		TradesExecuted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_trades_executed_total",
			Help: "Total trade executions.",
		}),
		IdempotencyReplays: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "idempotency_replays_total",
			Help: "Total idempotent order create replays.",
		}),
		KafkaPublishErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_publish_errors_total",
			Help: "Total Kafka publish errors.",
		}),
		ProcessingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "order_processing_duration_seconds",
			Help:    "Order processing duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}),
		OutboxClaimed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_outbox_claimed_total",
			Help: "Total outbox rows claimed for publishing.",
		}),
		OutboxPublished: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_outbox_published_total",
			Help: "Total outbox events published successfully.",
		}, []string{"event_type", "topic"}),
		OutboxPublishErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_outbox_publish_errors_total",
			Help: "Total outbox publish errors.",
		}, []string{"event_type", "topic", "result"}),
		OutboxTerminalFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_outbox_terminal_failures_total",
			Help: "Total outbox events moved to terminal failure.",
		}, []string{"event_type", "topic"}),
		OutboxPending: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tradeops_outbox_pending",
			Help: "Current pending outbox rows.",
		}),
		OutboxProcessing: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tradeops_outbox_processing",
			Help: "Current processing outbox rows.",
		}),
		OutboxFailed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tradeops_outbox_failed",
			Help: "Current terminally failed outbox rows.",
		}),
		OutboxOldestPendingAge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tradeops_outbox_oldest_pending_age_seconds",
			Help: "Age of the oldest pending outbox row in seconds.",
		}),
		OutboxPublishDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "tradeops_outbox_publish_duration_seconds",
			Help:    "Outbox Kafka publish duration in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}, []string{"event_type", "topic", "result"}),
		OutboxBatchSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tradeops_outbox_batch_size",
			Help:    "Outbox claimed batch size.",
			Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 250},
		}),
		OutboxClaimDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tradeops_outbox_claim_duration_seconds",
			Help:    "Outbox claim duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),
		OutboxRetryDelay: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tradeops_outbox_retry_delay_seconds",
			Help:    "Outbox retry delay in seconds.",
			Buckets: []float64{1, 2, 5, 10, 30, 60, 120, 300},
		}),
		OutboxLeaseRecoveries: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_outbox_lease_recoveries_total",
			Help: "Total outbox rows recovered after an expired lease.",
		}),
		OrderExpiryPolls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_order_expiry_polls_total",
			Help: "Total order expiry worker polls.",
		}, []string{"result"}),
		OrderExpiryDueOrders: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tradeops_order_expiry_due_orders",
			Help: "Current due orders visible to the expiry worker.",
		}),
		OrderExpiryClaimed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_order_expiry_claimed_total",
			Help: "Total orders claimed by the expiry worker.",
		}),
		OrdersExpired: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_orders_expired_total",
			Help: "Total orders expired by the expiry worker.",
		}, []string{"time_in_force", "reason"}),
		OrderExpiryErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_order_expiry_errors_total",
			Help: "Total order expiry worker errors.",
		}, []string{"result"}),
		OrderExpiryConflicts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_order_expiry_conflicts_total",
			Help: "Total order expiry no-op conflicts.",
		}),
		OrderExpiryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "tradeops_order_expiry_processing_duration_seconds",
			Help:    "Order expiry processing duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}, []string{"result"}),
		OrderExpiryBatchSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tradeops_order_expiry_batch_size",
			Help:    "Order expiry claimed batch size.",
			Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 250, 500},
		}),
		OrderExpiryLag: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tradeops_order_expiry_lag_seconds",
			Help:    "Seconds between configured expiry and actual expiry.",
			Buckets: []float64{0, 1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		}),
		OrderExpiryOldestDueAge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tradeops_order_expiry_oldest_due_age_seconds",
			Help: "Age of the oldest due order in seconds.",
		}),
		OrderExpiryReconciliationErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_order_expiry_reconciliation_errors_total",
			Help: "Total malformed order records skipped by expiry reconciliation.",
		}),
	}
	registry.MustRegister(metrics.OrdersCreated, metrics.OrdersAccepted, metrics.OrdersFilled, metrics.OrdersRejected, metrics.OrdersCancelled, metrics.OrdersAmended, metrics.OrdersPartial, metrics.TradesExecuted, metrics.IdempotencyReplays, metrics.KafkaPublishErrors, metrics.ProcessingDuration, metrics.OutboxClaimed, metrics.OutboxPublished, metrics.OutboxPublishErrors, metrics.OutboxTerminalFailures, metrics.OutboxPending, metrics.OutboxProcessing, metrics.OutboxFailed, metrics.OutboxOldestPendingAge, metrics.OutboxPublishDuration, metrics.OutboxBatchSize, metrics.OutboxClaimDuration, metrics.OutboxRetryDelay, metrics.OutboxLeaseRecoveries, metrics.OrderExpiryPolls, metrics.OrderExpiryDueOrders, metrics.OrderExpiryClaimed, metrics.OrdersExpired, metrics.OrderExpiryErrors, metrics.OrderExpiryConflicts, metrics.OrderExpiryDuration, metrics.OrderExpiryBatchSize, metrics.OrderExpiryLag, metrics.OrderExpiryOldestDueAge, metrics.OrderExpiryReconciliationErrors)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveProcessing(start time.Time) {
	m.ProcessingDuration.Observe(time.Since(start).Seconds())
}
