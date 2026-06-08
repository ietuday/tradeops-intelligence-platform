package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/repository"
)

func TestNormalizeOrderCreatedEvent(t *testing.T) {
	event, err := NormalizeEvent("order.created", payload(t, map[string]any{
		"eventId": "event-1",
		"orderId": "order-1",
		"userId":  "user-1",
	}), "corr-1")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if event.ServiceName != "order-service" || event.Action != "ORDER_CREATED" || ptrValue(event.EntityID) != "order-1" || ptrValue(event.ActorUserID) != "user-1" || event.Severity != domain.SeverityInfo {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestNormalizeSurveillanceAlertCreatedEvent(t *testing.T) {
	event, err := NormalizeEvent("surveillance.alert.created", payload(t, map[string]any{
		"eventId":  "event-1",
		"alertId":  "alert-1",
		"userId":   "user-1",
		"severity": "CRITICAL",
	}), "")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if event.ServiceName != "surveillance-service" || event.Action != "SURVEILLANCE_ALERT_CREATED" || ptrValue(event.EntityID) != "alert-1" || event.Severity != domain.SeverityHigh {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestNormalizeNotificationReadEvent(t *testing.T) {
	event, err := NormalizeEvent("notification.read", payload(t, map[string]any{
		"eventId":        "event-1",
		"notificationId": "notification-1",
		"userId":         "user-1",
	}), "")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if event.ServiceName != "notification-service" || event.Action != "NOTIFICATION_READ" || event.Severity != domain.SeverityInfo {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestNormalizeRiskBreachedEvent(t *testing.T) {
	event, err := NormalizeEvent("risk.breached", payload(t, map[string]any{
		"eventId":     "event-1",
		"portfolioId": "portfolio-1",
		"userId":      "user-1",
	}), "")
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if event.ServiceName != "risk-engine-service" || event.Action != "RISK_BREACHED" || event.Severity != domain.SeverityHigh {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestNormalizeBadPayloadReturnsError(t *testing.T) {
	if _, err := NormalizeEvent("order.created", []byte("{bad"), ""); err == nil {
		t.Fatal("expected bad payload error")
	}
}

func TestProcessEventSkipsDuplicateAuditLog(t *testing.T) {
	store := &fakeStore{createErr: repository.ErrDuplicate}
	svc := NewAuditService(store, observability.NewMetrics())

	err := svc.ProcessEvent(context.Background(), domain.SourceEvent{Topic: "order.created", Value: payload(t, map[string]any{
		"eventId": "event-1",
		"orderId": "order-1",
	})})
	if err != nil {
		t.Fatalf("duplicate should be skipped without error: %v", err)
	}
	if store.createdCount != 1 {
		t.Fatalf("expected one create attempt, got %d", store.createdCount)
	}
}

func TestCSVExportFormat(t *testing.T) {
	store := &fakeStore{logs: []domain.AuditLog{sampleLog()}}
	svc := NewAuditService(store, observability.NewMetrics())

	result, err := svc.Export(context.Background(), riskManagerUser(), repository.ListFilters{Limit: 50}, "csv")
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if result.ContentType != "text/csv" || !strings.Contains(string(result.Body), "event_type") || !strings.Contains(string(result.Body), "ORDER_CREATED") {
		t.Fatalf("unexpected csv export: %s", string(result.Body))
	}
}

func TestSummary(t *testing.T) {
	store := &fakeStore{summary: repository.Summary{Total: 2, Last24Hours: 1, ByService: map[string]int64{"order-service": 2}}}
	svc := NewAuditService(store, observability.NewMetrics())

	summary, err := svc.Summary(context.Background(), analystUser(), repository.ListFilters{})
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}
	if summary.Total != 2 || summary.ByService["order-service"] != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

type fakeStore struct {
	logs         []domain.AuditLog
	summary      repository.Summary
	createErr    error
	createdCount int
}

func (s *fakeStore) CreateAuditLog(_ context.Context, log domain.AuditLog) (domain.AuditLog, error) {
	s.createdCount++
	if s.createErr != nil {
		return domain.AuditLog{}, s.createErr
	}
	s.logs = append(s.logs, log)
	return log, nil
}

func (s *fakeStore) GetAuditLogByID(context.Context, string) (domain.AuditLog, error) {
	if len(s.logs) == 0 {
		return domain.AuditLog{}, repository.ErrNotFound
	}
	return s.logs[0], nil
}

func (s *fakeStore) ListAuditLogs(context.Context, repository.ListFilters) ([]domain.AuditLog, error) {
	return s.logs, nil
}

func (s *fakeStore) GetAuditSummary(context.Context, repository.ListFilters) (repository.Summary, error) {
	return s.summary, nil
}

func (s *fakeStore) ExportAuditLogs(context.Context, repository.ListFilters) ([]domain.AuditLog, error) {
	return s.logs, nil
}

func (s *fakeStore) CreateExportRequest(context.Context, domain.ExportRequest) (domain.ExportRequest, error) {
	return domain.ExportRequest{}, nil
}

func payload(t *testing.T, value map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sampleLog() domain.AuditLog {
	userID := "user-1"
	entityType := "ORDER"
	entityID := "order-1"
	return domain.AuditLog{
		ID:          "audit-1",
		EventType:   "order.created",
		ServiceName: "order-service",
		ActorUserID: &userID,
		EntityType:  &entityType,
		EntityID:    &entityID,
		Action:      "ORDER_CREATED",
		Description: "order created",
		Severity:    domain.SeverityInfo,
		Metadata:    map[string]any{},
		CreatedAt:   time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
	}
}

func analystUser() UserContext {
	return UserContext{UserID: "analyst-1", Roles: []string{"analyst"}}
}

func riskManagerUser() UserContext {
	return UserContext{UserID: "risk-1", Roles: []string{"risk_manager"}}
}

func TestReadRequiresAuditRole(t *testing.T) {
	svc := NewAuditService(&fakeStore{}, observability.NewMetrics())
	_, err := svc.ListAuditLogs(context.Background(), UserContext{UserID: "trader-1", Roles: []string{"trader"}}, repository.ListFilters{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
