package domain

import "time"

const (
	SideBuy  = "BUY"
	SideSell = "SELL"

	OrderTypeMarket   = "MARKET"
	OrderTypeLimit    = "LIMIT"
	OrderTypeStopLoss = "STOP_LOSS"

	StatusCreated   = "created"
	StatusValidated = "validated"
	StatusAccepted  = "accepted"
	StatusFilled    = "filled"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
)

type Order struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenantId"`
	UserID        string     `json:"userId"`
	Symbol        string     `json:"symbol"`
	Side          string     `json:"side"`
	OrderType     string     `json:"orderType"`
	Quantity      float64    `json:"quantity"`
	LimitPrice    *float64   `json:"limitPrice"`
	StopPrice     *float64   `json:"stopPrice"`
	Status        string     `json:"status"`
	FillPrice     *float64   `json:"fillPrice"`
	RejectReason  *string    `json:"rejectReason"`
	CorrelationID string     `json:"correlationId"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	CancelledAt   *time.Time `json:"cancelledAt"`
	FilledAt      *time.Time `json:"filledAt"`
}

type CreateOrderRequest struct {
	Symbol     string   `json:"symbol"`
	Side       string   `json:"side"`
	OrderType  string   `json:"orderType"`
	Quantity   float64  `json:"quantity"`
	LimitPrice *float64 `json:"limitPrice"`
	StopPrice  *float64 `json:"stopPrice"`
}
