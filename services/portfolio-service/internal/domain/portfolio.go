package domain

import "time"

type Portfolio struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	CashBalance float64   `json:"cashBalance"`
	RealizedPnL float64   `json:"realizedPnl"`
	TotalValue  float64   `json:"totalValue"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
