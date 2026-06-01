package domain

import "time"

type Candle struct {
	ID          string
	Symbol      string
	Interval    string
	Open        float64
	High        float64
	Low         float64
	Close       float64
	Volume      float64
	CandleStart time.Time
	CandleEnd   time.Time
	CreatedAt   time.Time
}
