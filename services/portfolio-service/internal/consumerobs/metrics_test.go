package consumerobs

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetricsAreUpdatedWithBoundedLabels(t *testing.T) {
	metrics := NewMetrics()
	registry := prometheus.NewRegistry()
	metrics.Register(registry)

	metrics.ConsumerLagMessages.WithLabelValues("portfolio-service", "portfolio-service", "trade.executed", "0").Set(42)
	metrics.ConsumerLagOldestAge.WithLabelValues("portfolio-service", "portfolio-service", "trade.executed").Set(15)
	metrics.ConsumerProcessingTotal.WithLabelValues("portfolio-service", "portfolio-service", "trade.executed", "trade.executed", "success").Inc()
	metrics.ConsumerProcessingErrors.WithLabelValues("portfolio-service", "portfolio-service", "trade.executed", "trade.executed", "processing_error").Inc()
	metrics.DLQMessages.WithLabelValues("portfolio-service", "portfolio.dlq").Set(2)
	metrics.DLQOldestMessageAge.WithLabelValues("portfolio-service", "portfolio.dlq").Set(60)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	assertMetricValue(t, families, "tradeops_consumer_lag_messages", 42)
	assertMetricValue(t, families, "tradeops_consumer_lag_oldest_age_seconds", 15)
	assertMetricValue(t, families, "tradeops_consumer_processing_total", 1)
	assertMetricValue(t, families, "tradeops_consumer_processing_errors_total", 1)
	assertMetricValue(t, families, "tradeops_dlq_messages", 2)
	assertMetricValue(t, families, "tradeops_dlq_oldest_message_age_seconds", 60)
	assertNoLabel(t, families, "eventId")
	assertNoLabel(t, families, "orderId")
	assertNoLabel(t, families, "correlationId")
	assertNoLabel(t, families, "requestId")
}

func assertMetricValue(t *testing.T, families []*dto.MetricFamily, name string, want float64) {
	t.Helper()
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		if len(family.GetMetric()) == 0 {
			t.Fatalf("%s has no samples", name)
		}
		metric := family.GetMetric()[0]
		switch {
		case metric.Gauge != nil && metric.Gauge.GetValue() == want:
			return
		case metric.Counter != nil && metric.Counter.GetValue() == want:
			return
		}
		t.Fatalf("%s value = %v, want %v", name, metric, want)
	}
	t.Fatalf("metric %s not found", name)
}

func assertNoLabel(t *testing.T, families []*dto.MetricFamily, forbidden string) {
	t.Helper()
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == forbidden {
					t.Fatalf("forbidden label %s found on %s", forbidden, family.GetName())
				}
			}
		}
	}
}
