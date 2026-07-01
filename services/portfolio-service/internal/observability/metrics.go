package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry               *prometheus.Registry
	Updates                prometheus.Counter
	UpdateFailures         prometheus.Counter
	HoldingsCount          prometheus.Gauge
	CashBalance            prometheus.Gauge
	RealizedPnL            prometheus.Gauge
	UnrealizedPnL          prometheus.Gauge
	ProcessingDuration     prometheus.Histogram
	KafkaPublishErrors     prometheus.Counter
	EventsRetried          prometheus.CounterVec
	EventsDeadlettered     prometheus.CounterVec
	ProcessingAttempts     prometheus.CounterVec
	DuplicateSkipped       prometheus.CounterVec
	TradeEventsReceived    prometheus.CounterVec
	TradeEventsProcessed   prometheus.CounterVec
	TradeEventsFailed      prometheus.CounterVec
	TradeEventsDuplicate   prometheus.Counter
	TradePayloadConflicts  prometheus.Counter
	ExecutionLag           prometheus.Gauge
	ReconciliationFailures prometheus.CounterVec
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
		EventsRetried: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "portfolio_events_retried_total",
			Help: "Total portfolio source events retried.",
		}, []string{"topic"}),
		EventsDeadlettered: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "portfolio_events_deadlettered_total",
			Help: "Total portfolio source events published to DLQ.",
		}, []string{"topic"}),
		ProcessingAttempts: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "portfolio_event_processing_attempts_total",
			Help: "Total portfolio event processing attempts by status.",
		}, []string{"topic", "status"}),
		DuplicateSkipped: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "portfolio_duplicate_events_skipped_total",
			Help: "Total duplicate portfolio events skipped.",
		}, []string{"topic"}),
		TradeEventsReceived: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_portfolio_trade_events_received_total",
			Help: "Total trade.executed events received by Portfolio Service.",
		}, []string{"event_version"}),
		TradeEventsProcessed: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_portfolio_trade_events_processed_total",
			Help: "Total trade.executed events applied by Portfolio Service.",
		}, []string{"result"}),
		TradeEventsFailed: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_portfolio_trade_events_failed_total",
			Help: "Total trade.executed events that failed processing.",
		}, []string{"reason"}),
		TradeEventsDuplicate: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_portfolio_trade_events_duplicate_total",
			Help: "Total duplicate trade.executed events skipped.",
		}),
		TradePayloadConflicts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tradeops_portfolio_trade_payload_conflicts_total",
			Help: "Total conflicting payloads seen for an already processed execution ID.",
		}),
		ExecutionLag: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tradeops_portfolio_execution_lag_seconds",
			Help: "Lag between trade execution occurrence and portfolio processing.",
		}),
		ReconciliationFailures: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_portfolio_reconciliation_failures_total",
			Help: "Total portfolio reconciliation failures while processing executions.",
		}, []string{"reason"}),
	}
	registry.MustRegister(metrics.Updates, metrics.UpdateFailures, metrics.HoldingsCount, metrics.CashBalance, metrics.RealizedPnL, metrics.UnrealizedPnL, metrics.ProcessingDuration, metrics.KafkaPublishErrors, &metrics.EventsRetried, &metrics.EventsDeadlettered, &metrics.ProcessingAttempts, &metrics.DuplicateSkipped, &metrics.TradeEventsReceived, &metrics.TradeEventsProcessed, &metrics.TradeEventsFailed, metrics.TradeEventsDuplicate, metrics.TradePayloadConflicts, metrics.ExecutionLag, &metrics.ReconciliationFailures)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveProcessing(start time.Time) {
	m.ProcessingDuration.Observe(time.Since(start).Seconds())
}
