package domain

import "time"

type Holding struct {
	ID              string    `json:"id"`
	PortfolioID     string    `json:"portfolioId"`
	UserID          string    `json:"userId"`
	Symbol          string    `json:"symbol"`
	Quantity        float64   `json:"quantity"`
	AverageBuyPrice float64   `json:"averageBuyPrice"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
