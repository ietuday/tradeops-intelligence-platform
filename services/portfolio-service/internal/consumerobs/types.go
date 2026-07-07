package consumerobs

import "time"

const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusStalled  = "stalled"
	StatusUnknown  = "unknown"
)

type Config struct {
	Enabled              bool
	PollInterval         time.Duration
	LagWarnThreshold     int64
	LagCriticalThreshold int64
	StalledAfter         time.Duration
	MaxTopicScan         int
	DLQEnabled           bool
	DLQTopic             string
	DLQOldestAgeWarn     time.Duration
	DLQOldestAgeCritical time.Duration
}

type PartitionStatus struct {
	Partition     int   `json:"partition"`
	CurrentOffset int64 `json:"currentOffset"`
	LatestOffset  int64 `json:"latestOffset"`
	LagMessages   int64 `json:"lagMessages"`
}

type ConsumerStatus struct {
	ConsumerGroup       string            `json:"consumerGroup"`
	Topic               string            `json:"topic"`
	Partitions          []PartitionStatus `json:"partitions"`
	TotalLagMessages    int64             `json:"totalLagMessages"`
	OldestLagAgeSeconds float64           `json:"oldestLagAgeSeconds"`
	LastProcessedAt     *time.Time        `json:"lastProcessedAt,omitempty"`
	LastMessageAt       *time.Time        `json:"lastMessageAt,omitempty"`
	LastErrorAt         *time.Time        `json:"lastErrorAt,omitempty"`
	ConsecutiveErrors   int               `json:"consecutiveErrors"`
	Status              string            `json:"status"`
}

type DLQStatus struct {
	Topic                   string     `json:"topic"`
	MessageCount            int64      `json:"messageCount"`
	OldestMessageAgeSeconds float64    `json:"oldestMessageAgeSeconds"`
	NewestMessageAgeSeconds float64    `json:"newestMessageAgeSeconds"`
	LastObservedAt          *time.Time `json:"lastObservedAt,omitempty"`
	Status                  string     `json:"status"`
}

type Snapshot struct {
	Service   string           `json:"service"`
	Status    string           `json:"status"`
	CheckedAt time.Time        `json:"checkedAt"`
	Consumers []ConsumerStatus `json:"consumers"`
	DLQ       []DLQStatus      `json:"dlq"`
	Enabled   bool             `json:"enabled"`
}
