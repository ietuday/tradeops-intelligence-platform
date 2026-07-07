package consumerobs

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	ConsumerLagMessages      *prometheus.GaugeVec
	ConsumerLagOldestAge     *prometheus.GaugeVec
	ConsumerLastProcessed    *prometheus.GaugeVec
	ConsumerProcessingTotal  *prometheus.CounterVec
	ConsumerProcessingErrors *prometheus.CounterVec
	ConsumerRebalances       *prometheus.CounterVec
	DLQMessages              *prometheus.GaugeVec
	DLQOldestMessageAge      *prometheus.GaugeVec
	DLQEvents                *prometheus.CounterVec
	OutboxPendingEvents      *prometheus.GaugeVec
	OutboxOldestPendingAge   *prometheus.GaugeVec
	OutboxPublishErrors      *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		ConsumerLagMessages: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tradeops_consumer_lag_messages",
			Help: "Estimated consumer lag in messages.",
		}, []string{"service", "consumer_group", "topic", "partition"}),
		ConsumerLagOldestAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tradeops_consumer_lag_oldest_age_seconds",
			Help: "Estimated age of the oldest lagging message.",
		}, []string{"service", "consumer_group", "topic"}),
		ConsumerLastProcessed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tradeops_consumer_last_processed_timestamp_seconds",
			Help: "Unix timestamp of the last successfully processed message.",
		}, []string{"service", "consumer_group", "topic"}),
		ConsumerProcessingTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_consumer_processing_total",
			Help: "Consumer processing attempts by bounded status.",
		}, []string{"service", "consumer_group", "topic", "event_type", "status"}),
		ConsumerProcessingErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_consumer_processing_errors_total",
			Help: "Consumer processing errors by bounded reason.",
		}, []string{"service", "consumer_group", "topic", "event_type", "reason"}),
		ConsumerRebalances: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_consumer_rebalances_total",
			Help: "Consumer group rebalance observations.",
		}, []string{"service", "consumer_group", "topic"}),
		DLQMessages: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tradeops_dlq_messages",
			Help: "Estimated messages in DLQ topics.",
		}, []string{"service", "topic"}),
		DLQOldestMessageAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tradeops_dlq_oldest_message_age_seconds",
			Help: "Estimated oldest DLQ message age in seconds.",
		}, []string{"service", "topic"}),
		DLQEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_dlq_events_total",
			Help: "DLQ events by bounded event type and reason.",
		}, []string{"service", "topic", "event_type", "reason"}),
		OutboxPendingEvents: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tradeops_outbox_pending_events",
			Help: "Pending outbox events by service and event type.",
		}, []string{"service", "event_type"}),
		OutboxOldestPendingAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tradeops_outbox_oldest_pending_age_seconds",
			Help: "Oldest pending outbox age by service and event type.",
		}, []string{"service", "event_type"}),
		OutboxPublishErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tradeops_outbox_publish_errors_total",
			Help: "Outbox publish errors by service, event type, and bounded reason.",
		}, []string{"service", "event_type", "reason"}),
	}
}

func (m *Metrics) Register(registry *prometheus.Registry) {
	registry.MustRegister(
		m.ConsumerLagMessages,
		m.ConsumerLagOldestAge,
		m.ConsumerLastProcessed,
		m.ConsumerProcessingTotal,
		m.ConsumerProcessingErrors,
		m.ConsumerRebalances,
		m.DLQMessages,
		m.DLQOldestMessageAge,
		m.DLQEvents,
		m.OutboxPendingEvents,
		m.OutboxOldestPendingAge,
		m.OutboxPublishErrors,
	)
}
