package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/domain"
)

func NormalizeEvent(topic string, payload []byte, correlationID string) (domain.AuditEvent, error) {
	var data map[string]any
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &data); err != nil {
			return domain.AuditEvent{}, err
		}
	} else {
		data = map[string]any{}
	}
	event := baseEvent(topic, data, correlationID)
	switch topic {
	case "user.registered":
		apply(&event, "identity-service", "USER_REGISTERED", "USER", firstString(data, "userId", "id"), domain.SeverityInfo)
	case "user.login":
		apply(&event, "identity-service", "USER_LOGIN", "USER", firstString(data, "userId", "id"), domain.SeverityInfo)
	case "user.logout":
		apply(&event, "identity-service", "USER_LOGOUT", "USER", firstString(data, "userId", "id"), domain.SeverityInfo)
	case "order.created":
		apply(&event, "order-service", "ORDER_CREATED", "ORDER", firstString(data, "orderId", "id"), domain.SeverityInfo)
	case "order.cancelled":
		apply(&event, "order-service", "ORDER_CANCELLED", "ORDER", firstString(data, "orderId", "id"), domain.SeverityWarning)
	case "order.filled":
		apply(&event, "order-service", "ORDER_FILLED", "ORDER", firstString(data, "orderId", "id"), domain.SeverityInfo)
	case "portfolio.updated":
		apply(&event, "portfolio-service", "PORTFOLIO_UPDATED", "PORTFOLIO", firstString(data, "portfolioId", "id"), domain.SeverityInfo)
	case "risk.score.updated":
		severity := domain.SeverityInfo
		if number(data, "score") >= 80 {
			severity = domain.SeverityWarning
		}
		apply(&event, "risk-engine-service", "RISK_SCORE_UPDATED", "PORTFOLIO", firstString(data, "portfolioId", "id"), severity)
	case "risk.breached":
		apply(&event, "risk-engine-service", "RISK_BREACHED", "PORTFOLIO", firstString(data, "portfolioId", "id"), domain.SeverityHigh)
	case "surveillance.alert.created":
		severity := domain.SeverityWarning
		alertSeverity := strings.ToUpper(firstString(data, "severity"))
		if alertSeverity == domain.SeverityHigh || alertSeverity == domain.SeverityCritical {
			severity = domain.SeverityHigh
		}
		apply(&event, "surveillance-service", "SURVEILLANCE_ALERT_CREATED", "ALERT", firstString(data, "alertId", "id"), severity)
	case "surveillance.alert.acknowledged":
		apply(&event, "surveillance-service", "SURVEILLANCE_ALERT_ACKNOWLEDGED", "ALERT", firstString(data, "alertId", "id"), domain.SeverityInfo)
	case "surveillance.alert.resolved":
		apply(&event, "surveillance-service", "SURVEILLANCE_ALERT_RESOLVED", "ALERT", firstString(data, "alertId", "id"), domain.SeverityInfo)
	case "surveillance.alert.dismissed":
		apply(&event, "surveillance-service", "SURVEILLANCE_ALERT_DISMISSED", "ALERT", firstString(data, "alertId", "id"), domain.SeverityInfo)
	case "notification.read":
		apply(&event, "notification-service", "NOTIFICATION_READ", "NOTIFICATION", firstString(data, "notificationId", "id"), domain.SeverityInfo)
	case "notification.failed":
		apply(&event, "notification-service", "NOTIFICATION_FAILED", "NOTIFICATION", firstString(data, "notificationId", "id"), domain.SeverityWarning)
	case "notification.sent":
		apply(&event, "notification-service", "NOTIFICATION_SENT", "NOTIFICATION", firstString(data, "notificationId", "id"), domain.SeverityInfo)
	case "notification.retry_requested":
		apply(&event, "notification-service", "NOTIFICATION_RETRY_REQUESTED", "NOTIFICATION", firstString(data, "notificationId", "id"), domain.SeverityWarning)
	default:
		apply(&event, "unknown-service", "EVENT_OBSERVED", "EVENT", topic, domain.SeverityInfo)
	}
	if event.Action == "" {
		return domain.AuditEvent{}, errors.New("audit event missing action")
	}
	event.ActorUserID = stringPtr(firstString(data, "userId", "actorUserId", "actor_user_id"))
	event.ActorRole = stringPtr(firstString(data, "role", "actorRole", "actor_role"))
	event.IPAddress = stringPtr(firstString(data, "ipAddress", "ip_address"))
	event.UserAgent = stringPtr(firstString(data, "userAgent", "user_agent"))
	if event.Description == "" {
		event.Description = description(event.Action, event.EntityType, event.EntityID)
	}
	if event.SourceEventKey == nil {
		key := sourceEventKey(topic, event.Action, ptrValue(event.EntityID), firstString(data, "eventId", "id"), event.CreatedAt)
		event.SourceEventKey = &key
	}
	return event, nil
}

func baseEvent(topic string, data map[string]any, correlationID string) domain.AuditEvent {
	if correlationID == "" {
		correlationID = firstString(data, "correlationId", "correlation_id")
	}
	createdAt := parseTime(firstString(data, "occurredAt", "createdAt", "timestamp"))
	return domain.AuditEvent{
		EventType:     topic,
		CorrelationID: stringPtr(correlationID),
		Metadata: map[string]any{
			"originalPayload": data,
		},
		CreatedAt: createdAt,
	}
}

func apply(event *domain.AuditEvent, serviceName, action, entityType, entityID, severity string) {
	event.ServiceName = serviceName
	event.Action = action
	event.EntityType = stringPtr(entityType)
	event.EntityID = stringPtr(entityID)
	event.Severity = severity
}

func sourceEventKey(topic, action, entityID, eventID string, createdAt time.Time) string {
	if eventID != "" {
		return fmt.Sprintf("%s:%s", topic, eventID)
	}
	return fmt.Sprintf("%s:%s:%s:%s", topic, action, entityID, createdAt.Format(time.RFC3339Nano))
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case fmt.Stringer:
				return typed.String()
			}
		}
	}
	return ""
}

func number(data map[string]any, key string) float64 {
	if value, ok := data[key]; ok {
		switch typed := value.(type) {
		case float64:
			return typed
		case int:
			return float64(typed)
		}
	}
	return 0
}

func parseTime(value string) time.Time {
	if value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed
		}
	}
	return time.Now().UTC()
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
