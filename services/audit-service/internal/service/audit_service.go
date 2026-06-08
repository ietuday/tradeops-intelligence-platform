package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/repository"
)

var ErrForbidden = errors.New("forbidden")
var ErrInvalidExportFormat = errors.New("invalid export format")

type UserContext struct {
	UserID string
	Roles  []string
}

type Store interface {
	CreateAuditLog(context.Context, domain.AuditLog) (domain.AuditLog, error)
	GetAuditLogByID(context.Context, string) (domain.AuditLog, error)
	ListAuditLogs(context.Context, repository.ListFilters) ([]domain.AuditLog, error)
	GetAuditSummary(context.Context, repository.ListFilters) (repository.Summary, error)
	ExportAuditLogs(context.Context, repository.ListFilters) ([]domain.AuditLog, error)
	CreateExportRequest(context.Context, domain.ExportRequest) (domain.ExportRequest, error)
}

type Publisher interface {
	Publish(context.Context, domain.AuditLogEvent) error
}

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, domain.AuditLogEvent) error { return nil }

type AuditService struct {
	store     Store
	metrics   *observability.Metrics
	publisher Publisher
	logger    *slog.Logger
}

type ExportResult struct {
	ContentType string
	FileName    string
	Body        []byte
	Count       int
}

func NewAuditService(store Store, metrics *observability.Metrics) *AuditService {
	return NewAuditServiceWithPublisher(store, metrics, noopPublisher{}, slog.Default())
}

func NewAuditServiceWithPublisher(store Store, metrics *observability.Metrics, publisher Publisher, logger *slog.Logger) *AuditService {
	if publisher == nil {
		publisher = noopPublisher{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &AuditService{store: store, metrics: metrics, publisher: publisher, logger: logger}
}

func (s *AuditService) ProcessEvent(ctx context.Context, event domain.SourceEvent) error {
	auditEvent, err := s.NormalizeEvent(event.Topic, event.Value, event.CorrelationID)
	if err != nil {
		s.metrics.EventsFailed.WithLabelValues(event.Topic).Inc()
		return err
	}
	log, err := s.CreateAuditLog(ctx, auditEvent)
	if errors.Is(err, repository.ErrDuplicate) {
		s.metrics.DuplicatesSkipped.WithLabelValues(event.Topic).Inc()
		s.metrics.EventsProcessed.WithLabelValues(event.Topic).Inc()
		return nil
	}
	if err != nil {
		s.metrics.EventsFailed.WithLabelValues(event.Topic).Inc()
		return err
	}
	s.metrics.EventsProcessed.WithLabelValues(event.Topic).Inc()
	s.metrics.LogsCreated.WithLabelValues(log.ServiceName, log.EventType, log.Severity).Inc()
	return nil
}

func (s *AuditService) NormalizeEvent(topic string, payload []byte, correlationID string) (domain.AuditEvent, error) {
	return NormalizeEvent(topic, payload, correlationID)
}

func (s *AuditService) CreateAuditLog(ctx context.Context, event domain.AuditEvent) (domain.AuditLog, error) {
	log := domain.AuditLog{
		ID:             uuid.NewString(),
		EventType:      event.EventType,
		ServiceName:    event.ServiceName,
		ActorUserID:    event.ActorUserID,
		ActorRole:      event.ActorRole,
		EntityType:     event.EntityType,
		EntityID:       event.EntityID,
		Action:         event.Action,
		Description:    event.Description,
		Severity:       event.Severity,
		CorrelationID:  event.CorrelationID,
		IPAddress:      event.IPAddress,
		UserAgent:      event.UserAgent,
		Metadata:       event.Metadata,
		SourceEventKey: event.SourceEventKey,
		CreatedAt:      event.CreatedAt,
	}
	created, err := s.store.CreateAuditLog(ctx, log)
	if err != nil {
		return domain.AuditLog{}, err
	}
	if err := s.publisher.Publish(ctx, domain.AuditLogEvent{
		EventID:       uuid.NewString(),
		EventType:     "audit.log.created",
		AuditLogID:    created.ID,
		ServiceName:   created.ServiceName,
		Action:        created.Action,
		Severity:      created.Severity,
		EntityType:    created.EntityType,
		EntityID:      created.EntityID,
		ActorUserID:   created.ActorUserID,
		CorrelationID: created.CorrelationID,
		Metadata:      created.Metadata,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
		s.logger.Warn("failed to publish audit log event", "error", err)
	}
	return created, nil
}

func (s *AuditService) ListAuditLogs(ctx context.Context, user UserContext, filters repository.ListFilters) ([]domain.AuditLog, error) {
	if !canRead(user.Roles) {
		return nil, ErrForbidden
	}
	return s.store.ListAuditLogs(ctx, filters)
}

func (s *AuditService) GetAuditLog(ctx context.Context, user UserContext, id string) (domain.AuditLog, error) {
	if !canRead(user.Roles) {
		return domain.AuditLog{}, ErrForbidden
	}
	return s.store.GetAuditLogByID(ctx, id)
}

func (s *AuditService) Summary(ctx context.Context, user UserContext, filters repository.ListFilters) (repository.Summary, error) {
	if !canRead(user.Roles) {
		return repository.Summary{}, ErrForbidden
	}
	return s.store.GetAuditSummary(ctx, filters)
}

func (s *AuditService) Export(ctx context.Context, user UserContext, filters repository.ListFilters, format string) (ExportResult, error) {
	if !canExport(user.Roles) {
		return ExportResult{}, ErrForbidden
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "json"
	}
	logs, err := s.store.ExportAuditLogs(ctx, filters)
	if err != nil {
		return ExportResult{}, err
	}
	requestedBy := nullableUser(user.UserID)
	_, _ = s.store.CreateExportRequest(ctx, domain.ExportRequest{
		RequestedBy: requestedBy,
		Filters:     filtersMap(filters, format),
		Status:      "COMPLETED",
		RecordCount: len(logs),
	})
	s.metrics.ExportRequests.WithLabelValues(format).Inc()

	switch format {
	case "json":
		body, err := json.Marshal(map[string]any{"auditLogs": logs, "count": len(logs)})
		if err != nil {
			return ExportResult{}, err
		}
		return ExportResult{ContentType: "application/json", FileName: "audit-export.json", Body: body, Count: len(logs)}, nil
	case "csv":
		body, err := csvExport(logs)
		if err != nil {
			return ExportResult{}, err
		}
		return ExportResult{ContentType: "text/csv", FileName: "audit-export.csv", Body: body, Count: len(logs)}, nil
	default:
		return ExportResult{}, ErrInvalidExportFormat
	}
}

func csvExport(logs []domain.AuditLog) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"id", "event_type", "service_name", "actor_user_id", "entity_type", "entity_id", "action", "severity", "correlation_id", "created_at", "description"}); err != nil {
		return nil, err
	}
	for _, log := range logs {
		if err := writer.Write([]string{
			log.ID,
			log.EventType,
			log.ServiceName,
			ptrValue(log.ActorUserID),
			ptrValue(log.EntityType),
			ptrValue(log.EntityID),
			log.Action,
			log.Severity,
			ptrValue(log.CorrelationID),
			log.CreatedAt.Format(time.RFC3339),
			log.Description,
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

func canRead(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager", "analyst":
			return true
		}
	}
	return false
}

func canExport(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager":
			return true
		}
	}
	return false
}

func nullableUser(userID string) *string {
	if userID == "" {
		return nil
	}
	return &userID
}

func filtersMap(filters repository.ListFilters, format string) map[string]any {
	return map[string]any{
		"eventType":     filters.EventType,
		"serviceName":   filters.ServiceName,
		"actorUserId":   filters.ActorUserID,
		"entityType":    filters.EntityType,
		"entityId":      filters.EntityID,
		"action":        filters.Action,
		"severity":      filters.Severity,
		"correlationId": filters.CorrelationID,
		"format":        format,
	}
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func description(action string, entityType *string, entityID *string) string {
	if entityType == nil || entityID == nil || *entityID == "" {
		return strings.ReplaceAll(strings.ToLower(action), "_", " ")
	}
	return fmt.Sprintf("%s %s %s", strings.ReplaceAll(strings.ToLower(action), "_", " "), *entityType, *entityID)
}
