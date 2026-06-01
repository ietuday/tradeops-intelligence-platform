package domain

import "time"

type OrderFilledEvent struct {
	EventID       string    `json:"eventId"`
	EventType     string    `json:"eventType"`
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

type PortfolioEvent struct {
	EventID       string    `json:"eventId"`
	EventType     string    `json:"eventType"`
	PortfolioID   string    `json:"portfolioId"`
	UserID        string    `json:"userId"`
	CashBalance   float64   `json:"cashBalance"`
	TotalValue    float64   `json:"totalValue"`
	RealizedPnL   float64   `json:"realizedPnl"`
	OccurredAt    time.Time `json:"occurredAt"`
	CorrelationID string    `json:"correlationId"`
}
