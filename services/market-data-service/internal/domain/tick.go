package domain

import "time"

type Tick struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Volume        float64   `json:"volume"`
	Source        string    `json:"source"`
	EventTime     time.Time `json:"eventTime"`
	ReceivedAt    time.Time `json:"receivedAt"`
	CorrelationID string    `json:"correlationId"`
}

type NormalizedTickEvent struct {
	EventID       string    `json:"eventId"`
	EventType     string    `json:"eventType"`
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Volume        float64   `json:"volume"`
	Source        string    `json:"source"`
	EventTime     time.Time `json:"eventTime"`
	ReceivedAt    time.Time `json:"receivedAt"`
	CorrelationID string    `json:"correlationId"`
}
