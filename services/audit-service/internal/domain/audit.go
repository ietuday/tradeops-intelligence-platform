package domain

import "time"

const (
	SeverityInfo     = "INFO"
	SeverityWarning  = "WARNING"
	SeverityHigh     = "HIGH"
	SeverityCritical = "CRITICAL"
)

type AuditLog struct {
	ID             string         `json:"id"`
	EventType      string         `json:"eventType"`
	ServiceName    string         `json:"serviceName"`
	ActorUserID    *string        `json:"actorUserId,omitempty"`
	ActorRole      *string        `json:"actorRole,omitempty"`
	EntityType     *string        `json:"entityType,omitempty"`
	EntityID       *string        `json:"entityId,omitempty"`
	Action         string         `json:"action"`
	Description    string         `json:"description"`
	Severity       string         `json:"severity"`
	CorrelationID  *string        `json:"correlationId,omitempty"`
	IPAddress      *string        `json:"ipAddress,omitempty"`
	UserAgent      *string        `json:"userAgent,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	SourceEventKey *string        `json:"sourceEventKey,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type AuditEvent struct {
	EventType      string
	ServiceName    string
	ActorUserID    *string
	ActorRole      *string
	EntityType     *string
	EntityID       *string
	Action         string
	Description    string
	Severity       string
	CorrelationID  *string
	IPAddress      *string
	UserAgent      *string
	Metadata       map[string]any
	SourceEventKey *string
	CreatedAt      time.Time
}

type ExportRequest struct {
	ID          string         `json:"id"`
	RequestedBy *string        `json:"requestedBy,omitempty"`
	Filters     map[string]any `json:"filters"`
	Status      string         `json:"status"`
	FileName    *string        `json:"fileName,omitempty"`
	RecordCount int            `json:"recordCount"`
	CreatedAt   time.Time      `json:"createdAt"`
}
