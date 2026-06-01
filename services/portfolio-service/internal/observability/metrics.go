package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry           *prometheus.Registry
	Updates            prometheus.Counter
	UpdateFailures     prometheus.Counter
	HoldingsCount      prometheus.Gauge
	CashBalance        prometheus.Gauge
	RealizedPnL        prometheus.Gauge
	UnrealizedPnL      prometheus.Gauge
	ProcessingDuration prometheus.Histogram
	KafkaPublishErrors prometheus.Counter
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metrics := &Metrics{
		Registry: registry,
		Updates: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "portfolio_updates_total",
			Help: "Total portfolio updates processed.",
		}),
		UpdateFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "portfolio_update_failures_total",
			Help: "Total portfolio update failures.",
		}),
		HoldingsCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "portfolio_holdings_count",
			Help: "Current holdings count for the last updated portfolio.",
		}),
		CashBalance: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "portfolio_cash_balance",
			Help: "Cash balance for the last updated portfolio.",
		}),
		RealizedPnL: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "portfolio_realized_pnl_total",
			Help: "Realized PnL for the last updated portfolio.",
		}),
		UnrealizedPnL: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "portfolio_unrealized_pnl_total",
			Help: "Unrealized PnL placeholder for the last updated portfolio.",
		}),
		ProcessingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "portfolio_event_processing_duration_seconds",
			Help:    "Portfolio event processing duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}),
		KafkaPublishErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "kafka_publish_errors_total",
			Help: "Total Kafka publish errors.",
		}),
	}
	registry.MustRegister(metrics.Updates, metrics.UpdateFailures, metrics.HoldingsCount, metrics.CashBalance, metrics.RealizedPnL, metrics.UnrealizedPnL, metrics.ProcessingDuration, metrics.KafkaPublishErrors)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveProcessing(start time.Time) {
	m.ProcessingDuration.Observe(time.Since(start).Seconds())
}
