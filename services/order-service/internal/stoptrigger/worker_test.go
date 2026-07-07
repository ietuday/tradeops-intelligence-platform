package stoptrigger

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

type fakeRepo struct {
	result BatchResult
	err    error
	calls  int
}

func (f *fakeRepo) TriggerDueStopOrders(_ context.Context, _ int, _ time.Duration, _ time.Time) (BatchResult, error) {
	f.calls++
	return f.result, f.err
}

func testWorker(repo *fakeRepo) (*Worker, *observability.Metrics) {
	metrics := observability.NewMetrics()
	worker := NewWorker(repo, metrics, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{
		Enabled:              true,
		PollInterval:         time.Hour,
		BatchSize:            10,
		MaxReferencePriceAge: time.Minute,
		ProcessingTimeout:    time.Second,
		ShutdownTimeout:      time.Second,
	})
	return worker, metrics
}

func TestWorkerTriggersEligibleOrder(t *testing.T) {
	repo := &fakeRepo{result: BatchResult{
		Processed: 1,
		Triggered: 1,
		TriggeredLabels: []TriggeredLabel{{
			Symbol:    "AAPL",
			Side:      domain.SideBuy,
			OrderType: domain.OrderTypeStop,
		}},
	}}
	worker, metrics := testWorker(repo)
	worker.Poll(context.Background())

	status := worker.Status()
	if status.TriggeredInLastPoll != 1 || status.ConsecutiveErrors != 0 {
		t.Fatalf("unexpected status: %#v", status)
	}
	if got := metricValue(t, metrics, "tradeops_stop_trigger_orders_triggered_total", map[string]string{"symbol": "AAPL", "side": domain.SideBuy, "order_type": domain.OrderTypeStop}); got != 1 {
		t.Fatalf("trigger metric = %v, want 1", got)
	}
}

func TestWorkerSkipsNoEligibleOrders(t *testing.T) {
	repo := &fakeRepo{result: BatchResult{Processed: 0, Triggered: 0}}
	worker, metrics := testWorker(repo)
	worker.Poll(context.Background())
	if got := metricValue(t, metrics, "tradeops_stop_trigger_polls_total", map[string]string{"status": "skipped"}); got != 1 {
		t.Fatalf("skipped polls = %v, want 1", got)
	}
}

func TestWorkerRecordsErrorMetricOnFailure(t *testing.T) {
	repo := &fakeRepo{err: errors.New("database unavailable")}
	worker, metrics := testWorker(repo)
	worker.Poll(context.Background())
	if got := metricValue(t, metrics, "tradeops_stop_trigger_errors_total", map[string]string{"reason": "db_error"}); got != 1 {
		t.Fatalf("error metric = %v, want 1", got)
	}
	if status := worker.Status(); status.ConsecutiveErrors != 1 {
		t.Fatalf("consecutive errors = %d, want 1", status.ConsecutiveErrors)
	}
}

func TestWorkerDoesNotTriggerSameOrderTwiceWhenRepoNoops(t *testing.T) {
	repo := &fakeRepo{result: BatchResult{Processed: 1, Skipped: 1}}
	worker, metrics := testWorker(repo)
	worker.Poll(context.Background())
	worker.Poll(context.Background())
	if got := metricValue(t, metrics, "tradeops_stop_trigger_orders_triggered_total", map[string]string{"symbol": "AAPL", "side": domain.SideBuy, "order_type": domain.OrderTypeStop}); got != 0 {
		t.Fatalf("trigger metric = %v, want 0", got)
	}
}

func TestWorkerHandlesContextCancellation(t *testing.T) {
	repo := &fakeRepo{}
	worker, _ := testWorker(repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := worker.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func metricValue(t *testing.T, metrics *observability.Metrics, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := metrics.Registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if !labelsMatch(metric.GetLabel(), labels) {
				continue
			}
			if metric.GetCounter() != nil {
				return metric.GetCounter().GetValue()
			}
			if metric.GetGauge() != nil {
				return metric.GetGauge().GetValue()
			}
		}
	}
	return 0
}

func labelsMatch(pairs []*io_prometheus_client.LabelPair, labels map[string]string) bool {
	for key, want := range labels {
		found := false
		for _, pair := range pairs {
			if pair.GetName() == key && pair.GetValue() == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
