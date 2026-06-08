package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/repository"
)

var ErrForbidden = errors.New("forbidden")

type UserContext struct {
	UserID string
	Roles  []string
}

type SurveillanceService struct {
	repo     *repository.AlertRepository
	producer *kafka.Producer
	metrics  *observability.Metrics
	engine   *RuleEngine
}

func NewSurveillanceService(repo *repository.AlertRepository, producer *kafka.Producer, metrics *observability.Metrics, engine *RuleEngine) *SurveillanceService {
	return &SurveillanceService{repo: repo, producer: producer, metrics: metrics, engine: engine}
}

func (s *SurveillanceService) ProcessEvent(ctx context.Context, event domain.SourceEvent) error {
	s.metrics.KafkaMessages.WithLabelValues(event.Topic).Inc()
	now := time.Now().UTC()
	alerts, executions := s.engine.Evaluate(event, now)
	for _, execution := range executions {
		s.metrics.RuleExecutions.WithLabelValues(execution.RuleName, execution.SourceTopic).Inc()
		s.metrics.RuleDuration.Observe(float64(execution.ExecutionTimeMS) / 1000)
		if execution.Matched {
			s.metrics.RuleMatches.WithLabelValues(execution.RuleName).Inc()
		}
		if err := s.repo.SaveExecution(ctx, execution); err != nil {
			return err
		}
	}
	for _, alert := range alerts {
		created, err := s.repo.CreateAlert(ctx, alert)
		if err != nil {
			return err
		}
		s.metrics.AlertsCreated.Inc()
		if err := s.publishAlertEvent(ctx, "surveillance.alert.created", created, event.CorrelationID); err != nil {
			s.metrics.KafkaPublishErrors.Inc()
		}
	}
	return nil
}

func (s *SurveillanceService) ListAlerts(ctx context.Context, user UserContext, filters repository.AlertFilters) ([]domain.Alert, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	return s.repo.ListAlerts(ctx, filters)
}

func (s *SurveillanceService) GetAlert(ctx context.Context, user UserContext, id string) (domain.Alert, error) {
	if !canView(user.Roles) {
		return domain.Alert{}, ErrForbidden
	}
	return s.repo.GetAlert(ctx, id)
}

func (s *SurveillanceService) Acknowledge(ctx context.Context, user UserContext, id, correlationID string) (domain.Alert, error) {
	return s.transition(ctx, user, id, domain.StatusAcknowledged, "surveillance.alert.acknowledged", correlationID)
}

func (s *SurveillanceService) Resolve(ctx context.Context, user UserContext, id, correlationID string) (domain.Alert, error) {
	return s.transition(ctx, user, id, domain.StatusResolved, "surveillance.alert.resolved", correlationID)
}

func (s *SurveillanceService) Dismiss(ctx context.Context, user UserContext, id, correlationID string) (domain.Alert, error) {
	return s.transition(ctx, user, id, domain.StatusDismissed, "surveillance.alert.dismissed", correlationID)
}

func (s *SurveillanceService) Summary(ctx context.Context, user UserContext) (repository.Summary, error) {
	if !canView(user.Roles) {
		return repository.Summary{}, ErrForbidden
	}
	return s.repo.Summary(ctx)
}

func (s *SurveillanceService) transition(ctx context.Context, user UserContext, id, status, eventType, correlationID string) (domain.Alert, error) {
	if !canManage(user.Roles) {
		return domain.Alert{}, ErrForbidden
	}
	alert, err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		return domain.Alert{}, err
	}
	switch status {
	case domain.StatusAcknowledged:
		s.metrics.AlertsAcknowledged.Inc()
	case domain.StatusResolved:
		s.metrics.AlertsResolved.Inc()
	case domain.StatusDismissed:
		s.metrics.AlertsDismissed.Inc()
	}
	if err := s.publishAlertEvent(ctx, eventType, alert, correlationID); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
	}
	return alert, nil
}

func (s *SurveillanceService) publishAlertEvent(ctx context.Context, eventType string, alert domain.Alert, correlationID string) error {
	return s.producer.Publish(ctx, domain.AlertEvent{
		EventID:       uuid.NewString(),
		EventType:     eventType,
		AlertID:       alert.ID,
		AlertType:     alert.AlertType,
		Severity:      alert.Severity,
		EntityType:    alert.EntityType,
		EntityID:      alert.EntityID,
		UserID:        alert.UserID,
		Symbol:        alert.Symbol,
		Status:        alert.Status,
		Metadata:      alert.Metadata,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
	})
}

func canView(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager", "analyst", "viewer":
			return true
		}
	}
	return false
}

func canManage(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trading_admin", "risk_manager":
			return true
		}
	}
	return false
}
