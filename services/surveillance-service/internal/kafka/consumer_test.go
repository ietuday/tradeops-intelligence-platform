package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

func TestProcessWithRetryPublishesDLQAfterExhaustion(t *testing.T) {
	writer := &fakeWriter{}
	processor := &fakeProcessor{err: errors.New("invalid source event")}
	consumer := &Consumer{
		service:     processor,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		dlqWriter:   writer,
		retryConfig: RetryConfig{MaxRetries: 2, Backoff: time.Millisecond, BackoffMultiplier: 1},
	}

	err := consumer.processWithRetry(context.Background(), domain.SourceEvent{
		Topic:         "order.created",
		Key:           "order-1",
		Value:         []byte(`{"bad":true}`),
		CorrelationID: "corr-1",
	})
	if err == nil {
		t.Fatal("expected processing error")
	}
	if processor.calls != 3 {
		t.Fatalf("expected initial attempt plus two retries, got %d", processor.calls)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("expected one DLQ message, got %d", len(writer.messages))
	}
	var event DLQEvent
	if err := json.Unmarshal(writer.messages[0].Value, &event); err != nil {
		t.Fatalf("decode DLQ payload: %v", err)
	}
	if event.OriginalTopic != "order.created" || event.ServiceName != "surveillance-service" || event.RetryCount != 2 || event.CorrelationID != "corr-1" {
		t.Fatalf("unexpected DLQ event: %+v", event)
	}
}

type fakeProcessor struct {
	calls int
	err   error
}

func (p *fakeProcessor) ProcessEvent(context.Context, domain.SourceEvent) error {
	p.calls++
	return p.err
}

type fakeWriter struct {
	messages []kafka.Message
}

func (w *fakeWriter) WriteMessages(_ context.Context, messages ...kafka.Message) error {
	w.messages = append(w.messages, messages...)
	return nil
}

func (w *fakeWriter) Close() error {
	return nil
}
