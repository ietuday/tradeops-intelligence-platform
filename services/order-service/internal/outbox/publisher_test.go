package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/config"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
)

type fakeRepo struct {
	claimed       []Record
	published     []string
	failed        []string
	terminal      []string
	nextAvailable time.Time
	lastError     string
}

func (f *fakeRepo) Claim(context.Context, string, int, int, time.Duration) ([]Record, int64, error) {
	return f.claimed, 0, nil
}

func (f *fakeRepo) MarkPublished(_ context.Context, id, _ string) (bool, error) {
	f.published = append(f.published, id)
	return true, nil
}

func (f *fakeRepo) MarkFailed(_ context.Context, id, _ string, nextAvailableAt time.Time, safeError string) (bool, error) {
	f.failed = append(f.failed, id)
	f.nextAvailable = nextAvailableAt
	f.lastError = safeError
	return true, nil
}

func (f *fakeRepo) MarkTerminal(_ context.Context, id, _, safeError string) (bool, error) {
	f.terminal = append(f.terminal, id)
	f.lastError = safeError
	return true, nil
}

func (f *fakeRepo) Snapshot(context.Context) (Stats, error) {
	return Stats{}, nil
}

type fakeProducer struct {
	mu      sync.Mutex
	err     error
	calls   int
	topic   string
	key     string
	value   string
	headers map[string]string
}

func (f *fakeProducer) PublishRaw(_ context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.topic = topic
	f.key = string(key)
	f.value = string(value)
	f.headers = headers
	return f.err
}

func TestHeadersAndMessageKey(t *testing.T) {
	record := sampleRecord()
	if got := messageKey(record); got != "tenant-a:order-1" {
		t.Fatalf("key = %s", got)
	}
	headers := headers(record)
	for _, key := range []string{"event-id", "event-type", "tenant-id", "correlation-id", "traceparent", "tracestate", "content-type"} {
		if headers[key] == "" {
			t.Fatalf("missing header %s", key)
		}
	}
}

func TestSanitizeText(t *testing.T) {
	got := sanitizeText("one\ntwo\rthree", 7)
	if got != "one two" {
		t.Fatalf("sanitizeText = %q", got)
	}
}

func sampleRecord() Record {
	return Record{
		ID:            "outbox-1",
		TenantID:      "tenant-a",
		AggregateType: "order",
		AggregateID:   "order-1",
		EventID:       "event-1",
		EventType:     "order.filled",
		Topic:         "order.filled",
		Payload:       []byte(`{"eventId":"event-1","eventType":"order.filled"}`),
		CorrelationID: "corr-1",
		TraceParent:   "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		TraceState:    "vendor=value",
		AttemptCount:  0,
	}
}

func testConfig() config.OutboxConfig {
	return config.OutboxConfig{
		Enabled:            true,
		PollInterval:       time.Millisecond,
		BatchSize:          10,
		PublishConcurrency: 1,
		PublishTimeout:     time.Second,
		ShutdownTimeout:    time.Second,
		LeaseDuration:      2 * time.Second,
		BaseBackoff:        time.Millisecond,
		MaxBackoff:         time.Second,
		MaxAttempts:        2,
		ErrorMaxLength:     64,
	}
}

func TestProducerFakeCapturesPublish(t *testing.T) {
	producer := &fakeProducer{}
	err := producer.PublishRaw(context.Background(), "order.filled", []byte("tenant-a:order-1"), []byte(`{}`), map[string]string{"event-type": "order.filled"})
	if err != nil {
		t.Fatal(err)
	}
	if producer.calls != 1 || producer.topic != "order.filled" || producer.key != "tenant-a:order-1" {
		t.Fatalf("unexpected publish capture: %#v", producer)
	}
}

func TestPublisherConstructorValidation(t *testing.T) {
	if _, err := NewPublisher(nil, &fakeProducer{}, observability.NewMetrics(), nil, testConfig()); err == nil {
		t.Fatal("expected missing repo to fail")
	}
	if _, err := NewPublisher(&fakeRepo{}, nil, observability.NewMetrics(), nil, testConfig()); err == nil {
		t.Fatal("expected missing producer to fail")
	}
	if _, err := NewBackoff(time.Second, time.Millisecond); err == nil {
		t.Fatal("expected bad backoff to fail")
	}
}

func TestPublishOneMarksPublished(t *testing.T) {
	repo := &fakeRepo{}
	producer := &fakeProducer{}
	publisher, err := NewPublisher(repo, producer, observability.NewMetrics(), nil, testConfig())
	if err != nil {
		t.Fatal(err)
	}

	publisher.publishOne(context.Background(), sampleRecord())

	if len(repo.published) != 1 || repo.published[0] != "outbox-1" {
		t.Fatalf("published marks = %#v", repo.published)
	}
	if producer.calls != 1 || producer.key != "tenant-a:order-1" {
		t.Fatalf("unexpected producer call: %#v", producer)
	}
}

func TestPublishOneSchedulesRetry(t *testing.T) {
	repo := &fakeRepo{}
	producer := &fakeProducer{err: errors.New("broker unavailable")}
	publisher, err := NewPublisher(repo, producer, observability.NewMetrics(), nil, testConfig())
	if err != nil {
		t.Fatal(err)
	}

	publisher.publishOne(context.Background(), sampleRecord())

	if len(repo.failed) != 1 || repo.nextAvailable.IsZero() {
		t.Fatalf("expected retry scheduling, failed=%#v next=%s", repo.failed, repo.nextAvailable)
	}
	if repo.lastError == "" {
		t.Fatal("expected safe error")
	}
}

func TestPublishOneTerminalAfterMaxAttempts(t *testing.T) {
	repo := &fakeRepo{}
	producer := &fakeProducer{err: errors.New("broker unavailable")}
	publisher, err := NewPublisher(repo, producer, observability.NewMetrics(), nil, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	record := sampleRecord()
	record.AttemptCount = 1

	publisher.publishOne(context.Background(), record)

	if len(repo.terminal) != 1 {
		t.Fatalf("expected terminal failure, got %#v", repo.terminal)
	}
}

func TestPublishOneMalformedPayloadIsTerminal(t *testing.T) {
	repo := &fakeRepo{}
	producer := &fakeProducer{}
	publisher, err := NewPublisher(repo, producer, observability.NewMetrics(), nil, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	record := sampleRecord()
	record.Payload = []byte(`{"broken"`)

	publisher.publishOne(context.Background(), record)

	if producer.calls != 0 {
		t.Fatal("malformed payload should not be published")
	}
	if len(repo.terminal) != 1 || repo.lastError == "" {
		t.Fatalf("expected terminal malformed payload, terminal=%#v error=%q", repo.terminal, repo.lastError)
	}
}

func TestResultLabel(t *testing.T) {
	if resultLabel(nil) != "success" {
		t.Fatal("nil error should be success")
	}
	if resultLabel(errors.New("boom")) != "error" {
		t.Fatal("non-nil error should be error")
	}
}
