package domain

import "time"

type OrderFilledEvent struct {
	EventID       string    `json:"eventId"`
	EventType     string    `json:"eventType"`
	EventVersion  string    `json:"eventVersion,omitempty"`
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

type TradeExecutedEvent struct {
	EventID           string    `json:"eventId"`
	EventType         string    `json:"eventType"`
	EventVersion      string    `json:"eventVersion,omitempty"`
	TenantID          string    `json:"tenantId"`
	CorrelationID     string    `json:"correlationId"`
	TraceParent       string    `json:"traceparent,omitempty"`
	OccurredAt        time.Time `json:"occurredAt"`
	ExecutionID       string    `json:"executionId"`
	BuyOrderID        string    `json:"buyOrderId"`
	SellOrderID       string    `json:"sellOrderId"`
	BuyerUserID       string    `json:"buyerUserId"`
	SellerUserID      string    `json:"sellerUserId"`
	Symbol            string    `json:"symbol"`
	ExecutionQuantity float64   `json:"executionQuantity"`
	ExecutionPrice    float64   `json:"executionPrice"`
	Currency          string    `json:"currency,omitempty"`
	Source            string    `json:"source,omitempty"`
}

type PortfolioEvent struct {
	EventID       string    `json:"eventId"`
	EventType     string    `json:"eventType"`
	EventVersion  string    `json:"eventVersion,omitempty"`
	TenantID      string    `json:"tenantId"`
	PortfolioID   string    `json:"portfolioId"`
	UserID        string    `json:"userId"`
	CashBalance   float64   `json:"cashBalance"`
	TotalValue    float64   `json:"totalValue"`
	RealizedPnL   float64   `json:"realizedPnl"`
	OccurredAt    time.Time `json:"occurredAt"`
	CorrelationID string    `json:"correlationId"`
}
