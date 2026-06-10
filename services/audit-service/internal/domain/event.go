package domain

type SourceEvent struct {
	Topic         string
	Key           string
	Value         []byte
	CorrelationID string
	TraceParent   string
}

type AuditLogEvent struct {
	EventID       string         `json:"eventId"`
	EventType     string         `json:"eventType"`
	EventVersion  string         `json:"eventVersion,omitempty"`
	TenantID      string         `json:"tenantId"`
	AuditLogID    string         `json:"auditLogId"`
	ServiceName   string         `json:"serviceName"`
	Action        string         `json:"action"`
	Severity      string         `json:"severity"`
	EntityType    *string        `json:"entityType,omitempty"`
	EntityID      *string        `json:"entityId,omitempty"`
	ActorUserID   *string        `json:"actorUserId,omitempty"`
	CorrelationID *string        `json:"correlationId,omitempty"`
	TraceParent   string         `json:"traceparent,omitempty"`
	TraceID       string         `json:"traceId,omitempty"`
	SpanID        string         `json:"spanId,omitempty"`
	Metadata      map[string]any `json:"metadata"`
	OccurredAt    string         `json:"occurredAt"`
}
