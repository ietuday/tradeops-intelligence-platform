package domain

import "time"

type OrderEvent struct {
	EventID       string    `json:"eventId"`
	EventType     string    `json:"eventType"`
	TenantID      string    `json:"tenantId"`
	OrderID       string    `json:"orderId"`
	UserID        string    `json:"userId"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	OrderType     string    `json:"orderType"`
	Quantity      float64   `json:"quantity"`
	Status        string    `json:"status"`
	FillPrice     *float64  `json:"fillPrice"`
	OccurredAt    time.Time `json:"occurredAt"`
	CorrelationID string    `json:"correlationId"`
}
