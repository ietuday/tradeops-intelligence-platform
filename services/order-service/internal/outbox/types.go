package outbox

import (
	"context"
	"encoding/json"
	"time"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusPublished  = "published"
	StatusFailed     = "failed"
)

type Record struct {
	ID            string
	TenantID      string
	AggregateType string
	AggregateID   string
	EventID       string
	EventType     string
	Topic         string
	Payload       json.RawMessage
	CorrelationID string
	TraceParent   string
	TraceState    string
	Status        string
	AttemptCount  int
	AvailableAt   time.Time
	CreatedAt     time.Time
	LockedBy      string
	LockedAt      *time.Time
	LeaseUntil    *time.Time
	PublishedAt   *time.Time
	FailedAt      *time.Time
	LastError     string
}

type Stats struct {
	Pending           int64
	Processing        int64
	Failed            int64
	OldestPendingAge  time.Duration
	ConsecutiveErrors int64
	LastPoll          *time.Time
	LastPublished     *time.Time
	LastError         string
}

type EventProducer interface {
	PublishRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error
}
