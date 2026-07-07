package outbox

import "time"

const (
	StatusPending    = "pending"
	StatusPublishing = "publishing"
	StatusPublished  = "published"
	StatusFailed     = "failed"
)

type Event struct {
	ID            int64
	EventID       string
	TenantID      string
	AggregateType string
	AggregateID   string
	EventType     string
	EventVersion  string
	Topic         string
	Payload       []byte
	Headers       map[string]string
	Status        string
	Attempts      int
	NextAttemptAt time.Time
	LastError     *string
	CreatedAt     time.Time
	PublishedAt   *time.Time
}

type Status struct {
	Enabled                 bool          `json:"enabled"`
	Running                 bool          `json:"running"`
	PollInterval            string        `json:"pollInterval"`
	BatchSize               int           `json:"batchSize"`
	Pending                 int64         `json:"pending"`
	Publishing              int64         `json:"publishing"`
	PublishedLastHour       int64         `json:"publishedLastHour"`
	Failed                  int64         `json:"failed"`
	OldestPendingAgeSeconds float64       `json:"oldestPendingAgeSeconds"`
	LastPublishedAt         *time.Time    `json:"lastPublishedAt,omitempty"`
	ConsecutiveErrors       int           `json:"consecutiveErrors"`
	LastError               string        `json:"lastError,omitempty"`
	OldestPendingAge        time.Duration `json:"-"`
}
