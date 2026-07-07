package idempotency

import (
	"strings"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
)

type ProcessedEvent struct {
	TenantID       string
	EventID        string
	EventType      string
	EventVersion   string
	SourceService  string
	AggregateID    string
	IdempotencyKey string
	CorrelationID  string
	ProcessedAt    time.Time
}

func BuildIdempotencyKey(event domain.TradeExecutedEvent) string {
	if value := strings.TrimSpace(event.ExecutionID); value != "" {
		return "execution:" + value
	}
	if value := strings.TrimSpace(event.TradeID); value != "" {
		return "trade:" + value
	}
	return "event:" + strings.TrimSpace(event.EventID)
}
