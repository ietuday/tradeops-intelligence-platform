package domain

import "time"

type Snapshot struct {
	ID            string    `json:"id"`
	PortfolioID   string    `json:"portfolioId"`
	TenantID      string    `json:"tenantId"`
	UserID        string    `json:"userId"`
	CashBalance   float64   `json:"cashBalance"`
	HoldingsValue float64   `json:"holdingsValue"`
	TotalValue    float64   `json:"totalValue"`
	RealizedPnL   float64   `json:"realizedPnl"`
	CreatedAt     time.Time `json:"createdAt"`
}
