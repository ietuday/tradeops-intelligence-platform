package outbox

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/observability"
)

type fakeStore struct {
	events        []Event
	published     int
	failed        int
	lastTerminal  bool
	lastNextRetry time.Time
}

func (s *fakeStore) FetchPending(context.Context, int) ([]Event, error) {
	events := s.events
	s.events = nil
	return events, nil
}

func (s *fakeStore) MarkPublished(context.Context, string, time.Time) error {
	s.published++
	return nil
}

func (s *fakeStore) MarkFailed(_ context.Context, _ string, _ error, next time.Time, terminal bool) error {
	s.failed++
	s.lastTerminal = terminal
	s.lastNextRetry = next
	return nil
}

func (s *fakeStore) Snapshot(context.Context) (Status, error) {
	return Status{Pending: int64(len(s.events))}, nil
}

type fakeProducer struct {
	err   error
	calls int
}

func (p *fakeProducer) PublishRaw(context.Context, string, []byte, []byte, map[string]string) error {
	p.calls++
	return p.err
}

func TestPublisherMarksSuccessPublished(t *testing.T) {
	store := &fakeStore{events: []Event{{EventID: "evt-1", TenantID: "tenant-1", AggregateID: "portfolio-1", EventType: "portfolio.updated", Topic: "portfolio.updated", Payload: []byte(`{}`)}}}
	producer := &fakeProducer{}
	publisher, err := NewPublisher(store, producer, observability.NewMetrics(), slog.New(slog.NewTextHandler(io.Discard, nil)), Config{Enabled: true, PollInterval: time.Hour, BatchSize: 10})
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	publisher.poll(context.Background())
	if producer.calls != 1 || store.published != 1 {
		t.Fatalf("expected one publish and mark, got calls=%d published=%d", producer.calls, store.published)
	}
}

func TestPublisherSchedulesRetryOnFailure(t *testing.T) {
	store := &fakeStore{events: []Event{{EventID: "evt-1", TenantID: "tenant-1", AggregateID: "portfolio-1", EventType: "portfolio.updated", Topic: "portfolio.updated", Payload: []byte(`{}`), Attempts: 1}}}
	producer := &fakeProducer{err: errors.New("broker unavailable")}
	publisher, err := NewPublisher(store, producer, observability.NewMetrics(), slog.New(slog.NewTextHandler(io.Discard, nil)), Config{Enabled: true, PollInterval: time.Hour, BatchSize: 10, MaxAttempts: 3, BaseBackoff: time.Second, MaxBackoff: time.Minute})
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	publisher.poll(context.Background())
	if store.failed != 1 || store.lastTerminal {
		t.Fatalf("expected retry failure, failed=%d terminal=%t", store.failed, store.lastTerminal)
	}
	if !store.lastNextRetry.After(time.Now().UTC()) {
		t.Fatalf("expected future retry time, got %s", store.lastNextRetry)
	}
}

func TestPublisherMarksTerminalAfterMaxAttempts(t *testing.T) {
	store := &fakeStore{events: []Event{{EventID: "evt-1", TenantID: "tenant-1", AggregateID: "portfolio-1", EventType: "portfolio.updated", Topic: "portfolio.updated", Payload: []byte(`{}`), Attempts: 2}}}
	producer := &fakeProducer{err: errors.New("broker unavailable")}
	publisher, err := NewPublisher(store, producer, observability.NewMetrics(), slog.New(slog.NewTextHandler(io.Discard, nil)), Config{Enabled: true, PollInterval: time.Hour, BatchSize: 10, MaxAttempts: 3})
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	publisher.poll(context.Background())
	if store.failed != 1 || !store.lastTerminal {
		t.Fatalf("expected terminal failure, failed=%d terminal=%t", store.failed, store.lastTerminal)
	}
}
