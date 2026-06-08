package domain

import "time"

const (
	StatusOpen         = "OPEN"
	StatusAcknowledged = "ACKNOWLEDGED"
	StatusResolved     = "RESOLVED"
	StatusDismissed    = "DISMISSED"
)

const (
	SeverityLow      = "LOW"
	SeverityMedium   = "MEDIUM"
	SeverityHigh     = "HIGH"
	SeverityCritical = "CRITICAL"
)

type Alert struct {
	ID             string         `json:"id"`
	AlertType      string         `json:"alertType"`
	Severity       string         `json:"severity"`
	EntityType     string         `json:"entityType"`
	EntityID       string         `json:"entityId"`
	UserID         *string        `json:"userId,omitempty"`
	Symbol         *string        `json:"symbol,omitempty"`
	Description    string         `json:"description"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"createdAt"`
	AcknowledgedAt *time.Time     `json:"acknowledgedAt,omitempty"`
	ResolvedAt     *time.Time     `json:"resolvedAt,omitempty"`
	DismissedAt    *time.Time     `json:"dismissedAt,omitempty"`
}

type AlertEvent struct {
	EventID       string         `json:"eventId"`
	EventType     string         `json:"eventType"`
	AlertID       string         `json:"alertId"`
	AlertType     string         `json:"alertType"`
	Severity      string         `json:"severity"`
	EntityType    string         `json:"entityType"`
	EntityID      string         `json:"entityId"`
	UserID        *string        `json:"userId,omitempty"`
	Symbol        *string        `json:"symbol,omitempty"`
	Status        string         `json:"status"`
	Metadata      map[string]any `json:"metadata"`
	OccurredAt    time.Time      `json:"occurredAt"`
	CorrelationID string         `json:"correlationId,omitempty"`
}
