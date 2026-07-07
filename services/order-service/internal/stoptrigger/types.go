package stoptrigger

import "time"

type Config struct {
	Enabled              bool
	PollInterval         time.Duration
	BatchSize            int
	MaxReferencePriceAge time.Duration
	ProcessingTimeout    time.Duration
	ShutdownTimeout      time.Duration
	MarketTopic          string
	ConsumerGroup        string
}

type Stats struct {
	Enabled              bool       `json:"enabled"`
	Running              bool       `json:"running"`
	PollInterval         string     `json:"pollInterval"`
	BatchSize            int        `json:"batchSize"`
	MaxReferencePriceAge string     `json:"maxReferencePriceAge"`
	LastPollAt           *time.Time `json:"lastPollAt,omitempty"`
	LastSuccessfulPollAt *time.Time `json:"lastSuccessfulPollAt,omitempty"`
	LastTriggeredAt      *time.Time `json:"lastTriggeredAt,omitempty"`
	TriggeredInLastPoll  int        `json:"triggeredInLastPoll"`
	ConsecutiveErrors    int        `json:"consecutiveErrors"`
	LastError            string     `json:"lastError,omitempty"`
}

type TriggerCandidate struct {
	OrderID        string
	TenantID       string
	UserID         string
	Symbol         string
	Side           string
	OriginalType   string
	StopPrice      float64
	LimitPrice     *float64
	ReferencePrice float64
	PriceUpdatedAt time.Time
	CorrelationID  string
}

type BatchResult struct {
	Processed       int
	Triggered       int
	Skipped         int
	LagSeconds      []float64
	TriggeredLabels []TriggeredLabel
}

type TriggeredLabel struct {
	Symbol    string
	Side      string
	OrderType string
}

type MarketPriceUpdated struct {
	EventType     string    `json:"eventType"`
	EventVersion  string    `json:"eventVersion"`
	TenantID      string    `json:"tenantId"`
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Source        string    `json:"source"`
	EventTime     time.Time `json:"eventTime"`
	CorrelationID string    `json:"correlationId"`
}

type ReferencePrice struct {
	TenantID  string
	Symbol    string
	Price     float64
	Source    string
	UpdatedAt time.Time
}
