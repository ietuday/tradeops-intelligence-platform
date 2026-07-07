package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/consumerobs"
	portfoliokafka "github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/kafka"
)

type fakeConsumer struct {
	snapshot consumerobs.Snapshot
}

func (f fakeConsumer) Status() portfoliokafka.Status {
	return portfoliokafka.Status{Running: true}
}

func (f fakeConsumer) ConsumerObsStatus() consumerobs.Snapshot {
	return f.snapshot
}

func TestConsumersStatusReturnsSnapshot(t *testing.T) {
	checkedAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	handler := NewHealthHandler(nil, nil, nil, fakeConsumer{snapshot: consumerobs.Snapshot{
		Service:   "portfolio-service",
		Status:    consumerobs.StatusDegraded,
		CheckedAt: checkedAt,
		Enabled:   true,
		Consumers: []consumerobs.ConsumerStatus{{
			ConsumerGroup:    "portfolio-service",
			Topic:            "trade.executed",
			TotalLagMessages: 25,
			Status:           consumerobs.StatusDegraded,
		}},
		DLQ: []consumerobs.DLQStatus{{
			Topic:        "portfolio.dlq",
			MessageCount: 1,
			Status:       consumerobs.StatusDegraded,
		}},
	}})

	recorder := httptest.NewRecorder()
	handler.ConsumersStatus(recorder, httptest.NewRequest(http.MethodGet, "/internal/consumers/status", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	var body consumerobs.Snapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Service != "portfolio-service" || body.Status != consumerobs.StatusDegraded || !body.Enabled {
		t.Fatalf("unexpected response: %+v", body)
	}
	if body.CheckedAt.Location() != time.UTC {
		t.Fatalf("checkedAt location = %v, want UTC", body.CheckedAt.Location())
	}
	if len(body.Consumers) != 1 || body.Consumers[0].Topic != "trade.executed" {
		t.Fatalf("consumer status missing: %+v", body.Consumers)
	}
	if len(body.DLQ) != 1 || body.DLQ[0].MessageCount != 1 {
		t.Fatalf("DLQ status missing: %+v", body.DLQ)
	}
	if strings.Contains(recorder.Body.String(), "secret") || strings.Contains(recorder.Body.String(), "payload") {
		t.Fatal("response exposed payload data")
	}
}

func TestConsumersStatusDisabledWhenConsumerMissing(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil, nil)
	recorder := httptest.NewRecorder()

	handler.ConsumersStatus(recorder, httptest.NewRequest(http.MethodGet, "/internal/consumers/status", nil))

	var body consumerobs.Snapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Enabled {
		t.Fatalf("enabled = true, want false")
	}
	if body.Status != consumerobs.StatusUnknown {
		t.Fatalf("status = %s, want unknown", body.Status)
	}
	if body.CheckedAt.Location() != time.UTC {
		t.Fatalf("checkedAt location = %v, want UTC", body.CheckedAt.Location())
	}
}
