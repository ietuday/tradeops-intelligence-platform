package domain

import "time"

type IdempotencyRecord struct {
	UserID      string
	Key         string
	RequestHash string
	OrderID     string
	CreatedAt   time.Time
}
