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
	TradeID           string    `json:"tradeId,omitempty"`
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
	EventID           string    `json:"eventId"`
	EventType         string    `json:"eventType"`
	EventVersion      string    `json:"eventVersion,omitempty"`
	TenantID          string    `json:"tenantId"`
	PortfolioID       string    `json:"portfolioId"`
	UserID            string    `json:"userId"`
	AccountID         string    `json:"accountId,omitempty"`
	Symbol            string    `json:"symbol,omitempty"`
	PositionQuantity  string    `json:"positionQuantity,omitempty"`
	AveragePrice      string    `json:"averagePrice,omitempty"`
	CashDelta         string    `json:"cashDelta,omitempty"`
	CashBalance       float64   `json:"cashBalance"`
	TotalValue        float64   `json:"totalValue"`
	RealizedPnL       float64   `json:"realizedPnl"`
	SourceEventID     string    `json:"sourceEventId,omitempty"`
	SourceExecutionID string    `json:"sourceExecutionId,omitempty"`
	UpdatedAt         time.Time `json:"updatedAt,omitempty"`
	OccurredAt        time.Time `json:"occurredAt"`
	CorrelationID     string    `json:"correlationId"`
}
