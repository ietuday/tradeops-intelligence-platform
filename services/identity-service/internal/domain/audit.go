package domain

import "time"

type AuditLog struct {
	ID            string
	UserID        *string
	Action        string
	CorrelationID string
	IPAddress     string
	UserAgent     string
	CreatedAt     time.Time
}
