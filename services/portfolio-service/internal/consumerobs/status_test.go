package consumerobs

import (
	"testing"
	"time"
)

func TestCalculateLag(t *testing.T) {
	if got := CalculateLag(1500, 1200); got != 300 {
		t.Fatalf("lag = %d, want 300", got)
	}
	if got := CalculateLag(1200, 1200); got != 0 {
		t.Fatalf("zero lag = %d", got)
	}
	if got := CalculateLag(10, -1); got != 10 {
		t.Fatalf("missing committed offset lag = %d, want 10", got)
	}
}

func TestClassifyConsumer(t *testing.T) {
	now := time.Now().UTC()
	recent := now.Add(-time.Second)
	old := now.Add(-3 * time.Minute)
	cfg := Config{LagWarnThreshold: 100, LagCriticalThreshold: 1000, StalledAfter: 2 * time.Minute}
	if got := ClassifyConsumer(10, 0, &recent, now, cfg); got != StatusHealthy {
		t.Fatalf("healthy status = %s", got)
	}
	if got := ClassifyConsumer(150, 0, &recent, now, cfg); got != StatusDegraded {
		t.Fatalf("degraded status = %s", got)
	}
	if got := ClassifyConsumer(10, 0, &old, now, cfg); got != StatusStalled {
		t.Fatalf("stalled status = %s", got)
	}
	if got := ClassifyConsumer(0, 3, &recent, now, cfg); got != StatusDegraded {
		t.Fatalf("error status = %s", got)
	}
}

func TestClassifyDLQ(t *testing.T) {
	cfg := Config{DLQOldestAgeCritical: 30 * time.Minute}
	if got := ClassifyDLQ(0, 0, cfg); got != StatusHealthy {
		t.Fatalf("empty DLQ = %s", got)
	}
	if got := ClassifyDLQ(1, time.Minute, cfg); got != StatusDegraded {
		t.Fatalf("non-empty DLQ = %s", got)
	}
	if got := ClassifyDLQ(1, time.Hour, cfg); got != "critical" {
		t.Fatalf("old DLQ = %s", got)
	}
}
