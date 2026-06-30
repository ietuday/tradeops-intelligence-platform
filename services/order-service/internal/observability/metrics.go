package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry           *prometheus.Registry
	OrdersCreated      prometheus.Counter
	OrdersAccepted     prometheus.Counter
	OrdersFilled       prometheus.Counter
	OrdersRejected     prometheus.Counter
	OrdersCancelled    prometheus.Counter
	OrdersAmended      prometheus.Counter
	OrdersPartial      prometheus.Counter
	TradesExecuted     prometheus.Counter
	IdempotencyReplays prometheus.Counter
	KafkaPublishErrors prometheus.Counter
	ProcessingDuration prometheus.Histogram
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
	}
	registry.MustRegister(metrics.OrdersCreated, metrics.OrdersAccepted, metrics.OrdersFilled, metrics.OrdersRejected, metrics.OrdersCancelled, metrics.OrdersAmended, metrics.OrdersPartial, metrics.TradesExecuted, metrics.IdempotencyReplays, metrics.KafkaPublishErrors, metrics.ProcessingDuration)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveProcessing(start time.Time) {
	m.ProcessingDuration.Observe(time.Since(start).Seconds())
}
