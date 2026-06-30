package domain

import "time"

type OrderEvent struct {
	EventID           string    `json:"eventId"`
	EventType         string    `json:"eventType"`
	EventVersion      string    `json:"eventVersion,omitempty"`
	TenantID          string    `json:"tenantId"`
	OrderID           string    `json:"orderId"`
	UserID            string    `json:"userId"`
	Symbol            string    `json:"symbol"`
	Side              string    `json:"side"`
	OrderType         string    `json:"orderType"`
	TimeInForce       string    `json:"timeInForce,omitempty"`
	Quantity          float64   `json:"quantity"`
	FilledQuantity    float64   `json:"filledQuantity,omitempty"`
	RemainingQuantity float64   `json:"remainingQuantity,omitempty"`
	ExecutionQuantity float64   `json:"executionQuantity,omitempty"`
	ExecutionPrice    *float64  `json:"executionPrice,omitempty"`
	AverageFillPrice  *float64  `json:"averageFillPrice,omitempty"`
	Status            string    `json:"status"`
	Version           int       `json:"version,omitempty"`
	BuyOrderID        string    `json:"buyOrderId,omitempty"`
	SellOrderID       string    `json:"sellOrderId,omitempty"`
	FillPrice         *float64  `json:"fillPrice"`
	OccurredAt        time.Time `json:"occurredAt"`
	CorrelationID     string    `json:"correlationId"`
	TraceParent       string    `json:"traceparent,omitempty"`
	TraceID           string    `json:"traceId,omitempty"`
	SpanID            string    `json:"spanId,omitempty"`
}
