package domain

import "time"

type SourceEvent struct {
	Topic         string
	Key           string
	Value         []byte
	CorrelationID string
	TraceParent   string
}

type SurveillanceAlertEvent struct {
	EventID       string         `json:"eventId"`
	EventType     string         `json:"eventType"`
	EventVersion  string         `json:"eventVersion,omitempty"`
	TenantID      string         `json:"tenantId"`
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
	TraceParent   string         `json:"traceparent,omitempty"`
}
