package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/repository"
)

var ErrForbidden = errors.New("forbidden")

type UserContext struct {
	UserID string
	Roles  []string
}

type PortfolioService struct {
	repo        *repository.PortfolioRepository
	producer    *kafka.Producer
	metrics     *observability.Metrics
	initialCash float64
}

func NewPortfolioService(repo *repository.PortfolioRepository, producer *kafka.Producer, metrics *observability.Metrics, initialCash float64) *PortfolioService {
	return &PortfolioService{repo: repo, producer: producer, metrics: metrics, initialCash: initialCash}
}

func (s *PortfolioService) ProcessOrderFilled(ctx context.Context, payload []byte) error {
	start := time.Now()
	defer s.metrics.ObserveProcessing(start)

	var event domain.OrderFilledEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		s.metrics.UpdateFailures.Inc()
		return err
	}
	if event.EventType != "order.filled" {
		return nil
	}
	result, err := s.repo.ApplyFilledOrder(ctx, event, s.initialCash)
	if err != nil {
		s.metrics.UpdateFailures.Inc()
		return err
	}
	if result.Duplicate {
		s.metrics.DuplicateSkipped.WithLabelValues(event.EventType).Inc()
		return nil
	}
	s.metrics.Updates.Inc()
	s.metrics.HoldingsCount.Set(float64(len(result.Holdings)))
	s.metrics.CashBalance.Set(result.Portfolio.CashBalance)
	s.metrics.RealizedPnL.Set(result.Portfolio.RealizedPnL)
	s.metrics.UnrealizedPnL.Set(0)

	portfolioEvent := domain.PortfolioEvent{
		EventID:       uuid.NewString(),
		EventType:     "portfolio.updated",
		PortfolioID:   result.Portfolio.ID,
		UserID:        result.Portfolio.UserID,
		CashBalance:   result.Portfolio.CashBalance,
		TotalValue:    result.Portfolio.TotalValue,
		RealizedPnL:   result.Portfolio.RealizedPnL,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: event.CorrelationID,
	}
	if err := s.producer.PublishPortfolioUpdated(ctx, portfolioEvent); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
	}
	if err := s.producer.PublishSnapshotCreated(ctx, result.Snapshot, event.CorrelationID); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
	}
	return nil
}

func (s *PortfolioService) Portfolio(ctx context.Context, user UserContext) (domain.Portfolio, error) {
	if !canView(user.Roles) {
		return domain.Portfolio{}, ErrForbidden
	}
	return s.repo.GetPortfolio(ctx, user.UserID, s.initialCash)
}

func (s *PortfolioService) Holdings(ctx context.Context, user UserContext) ([]domain.Holding, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	return s.repo.GetHoldings(ctx, user.UserID)
}

func (s *PortfolioService) Snapshots(ctx context.Context, user UserContext) ([]domain.Snapshot, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	return s.repo.GetSnapshots(ctx, user.UserID)
}

func (s *PortfolioService) PnL(ctx context.Context, user UserContext) (map[string]any, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	total, events, err := s.repo.GetRealizedPnL(ctx, user.UserID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"realizedPnl": total, "unrealizedPnl": 0, "events": events}, nil
}

func (s *PortfolioService) Exposure(ctx context.Context, user UserContext) (map[string]any, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	holdings, err := s.repo.GetHoldings(ctx, user.UserID)
	if err != nil {
		return nil, err
	}
	var total float64
	exposures := make([]map[string]any, 0, len(holdings))
	for _, holding := range holdings {
		value := holding.Quantity * holding.AverageBuyPrice
		total += value
		exposures = append(exposures, map[string]any{"symbol": holding.Symbol, "value": value, "quantity": holding.Quantity})
	}
	return map[string]any{"totalExposure": total, "exposures": exposures}, nil
}

func canView(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "trader", "trading_admin", "risk_manager", "analyst", "viewer":
			return true
		}
	}
	return false
}
