package domain

import "time"

type IdempotencyRecord struct {
	TenantID    string
	UserID      string
	Key         string
	RequestHash string
	OrderID     string
	CreatedAt   time.Time
}
