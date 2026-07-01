package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/repository"
)

var ErrForbidden = errors.New("forbidden")

type UserContext struct {
	UserID   string
	TenantID string
	Roles    []string
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
	var event domain.OrderFilledEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		s.metrics.UpdateFailures.Inc()
		return err
	}
	if event.EventType == "order.filled" {
		// trade.executed is authoritative for financial mutations. This legacy
		// event is consumed only so old topics can drain without double applying.
		s.metrics.DuplicateSkipped.WithLabelValues(event.EventType).Inc()
		return nil
	}
	return s.ProcessTradeExecuted(ctx, payload)
}

func (s *PortfolioService) ProcessTradeExecuted(ctx context.Context, payload []byte) error {
	start := time.Now()
	defer s.metrics.ObserveProcessing(start)

	var event domain.TradeExecutedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		s.metrics.UpdateFailures.Inc()
		return err
	}
	if event.EventType != "trade.executed" {
		return nil
	}
	if err := normalizeTradeExecuted(&event); err != nil {
		s.metrics.UpdateFailures.Inc()
		s.metrics.TradeEventsFailed.WithLabelValues("validation").Inc()
		return err
	}
	s.metrics.TradeEventsReceived.WithLabelValues(event.EventVersion).Inc()
	result, err := s.repo.ApplyTradeExecution(ctx, event, s.initialCash, repository.PayloadHash(event))
	if err != nil {
		s.metrics.UpdateFailures.Inc()
		if errors.Is(err, repository.ErrPayloadConflict) {
			s.metrics.TradePayloadConflicts.Inc()
		}
		if errors.Is(err, repository.ErrInsufficientCash) || errors.Is(err, repository.ErrInsufficientHoldings) || errors.Is(err, repository.ErrSelfTrade) {
			s.metrics.ReconciliationFailures.WithLabelValues(errorReason(err)).Inc()
		}
		s.metrics.TradeEventsFailed.WithLabelValues(errorReason(err)).Inc()
		return err
	}
	if result.Duplicate {
		s.metrics.DuplicateSkipped.WithLabelValues(event.EventType).Inc()
		s.metrics.TradeEventsDuplicate.Inc()
		return nil
	}
	s.metrics.Updates.Inc()
	s.metrics.TradeEventsProcessed.WithLabelValues("success").Inc()
	if !event.OccurredAt.IsZero() {
		s.metrics.ExecutionLag.Set(time.Since(event.OccurredAt).Seconds())
	}
	s.metrics.HoldingsCount.Set(float64(len(result.Holdings)))
	s.metrics.CashBalance.Set(result.Portfolio.CashBalance)
	s.metrics.RealizedPnL.Set(result.Portfolio.RealizedPnL)
	s.metrics.UnrealizedPnL.Set(0)

	portfolioEvent := domain.PortfolioEvent{
		EventID:       uuid.NewString(),
		EventType:     "portfolio.updated",
		TenantID:      event.TenantID,
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

func normalizeTradeExecuted(event *domain.TradeExecutedEvent) error {
	event.EventVersion = strings.TrimSpace(event.EventVersion)
	if event.EventVersion == "" {
		event.EventVersion = "1.0"
	}
	if event.EventVersion != "1.0" && event.EventVersion != "1" {
		return fmt.Errorf("%w: unsupported event version", repository.ErrInvalidExecution)
	}
	event.TenantID = strings.TrimSpace(event.TenantID)
	event.ExecutionID = strings.TrimSpace(event.ExecutionID)
	event.BuyOrderID = strings.TrimSpace(event.BuyOrderID)
	event.SellOrderID = strings.TrimSpace(event.SellOrderID)
	event.BuyerUserID = strings.TrimSpace(event.BuyerUserID)
	event.SellerUserID = strings.TrimSpace(event.SellerUserID)
	event.Symbol = strings.ToUpper(strings.TrimSpace(event.Symbol))
	event.Currency = strings.ToUpper(strings.TrimSpace(event.Currency))
	if event.Currency == "" {
		event.Currency = "USD"
	}
	if event.TenantID == "" || event.ExecutionID == "" || event.BuyOrderID == "" || event.SellOrderID == "" || event.BuyerUserID == "" || event.SellerUserID == "" || event.Symbol == "" {
		return fmt.Errorf("%w: required field missing", repository.ErrInvalidExecution)
	}
	if event.BuyOrderID == event.SellOrderID {
		return fmt.Errorf("%w: buy and sell order IDs match", repository.ErrInvalidExecution)
	}
	if event.BuyerUserID == event.SellerUserID {
		return repository.ErrSelfTrade
	}
	if event.ExecutionQuantity <= 0 {
		return fmt.Errorf("%w: quantity must be positive", repository.ErrInvalidExecution)
	}
	if event.ExecutionPrice <= 0 {
		return fmt.Errorf("%w: price must be positive", repository.ErrInvalidExecution)
	}
	if event.Currency != "USD" {
		return fmt.Errorf("%w: unsupported currency", repository.ErrInvalidExecution)
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurredAt is required", repository.ErrInvalidExecution)
	}
	return nil
}

func errorReason(err error) string {
	switch {
	case errors.Is(err, repository.ErrInsufficientCash):
		return "insufficient_cash"
	case errors.Is(err, repository.ErrInsufficientHoldings):
		return "insufficient_holdings"
	case errors.Is(err, repository.ErrPayloadConflict):
		return "payload_conflict"
	case errors.Is(err, repository.ErrSelfTrade):
		return "self_trade"
	case errors.Is(err, repository.ErrInvalidExecution):
		return "invalid_execution"
	default:
		return "repository_error"
	}
}

func (s *PortfolioService) Portfolio(ctx context.Context, user UserContext) (domain.Portfolio, error) {
	if !canView(user.Roles) {
		return domain.Portfolio{}, ErrForbidden
	}
	return s.repo.GetPortfolio(ctx, defaultTenant(user.TenantID), user.UserID, s.initialCash)
}

func (s *PortfolioService) Holdings(ctx context.Context, user UserContext) ([]domain.Holding, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	return s.repo.GetHoldings(ctx, defaultTenant(user.TenantID), user.UserID)
}

func (s *PortfolioService) Snapshots(ctx context.Context, user UserContext) ([]domain.Snapshot, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	return s.repo.GetSnapshots(ctx, defaultTenant(user.TenantID), user.UserID)
}

func (s *PortfolioService) PnL(ctx context.Context, user UserContext) (map[string]any, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	total, events, err := s.repo.GetRealizedPnL(ctx, defaultTenant(user.TenantID), user.UserID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"realizedPnl": total, "unrealizedPnl": 0, "events": events}, nil
}

func (s *PortfolioService) Exposure(ctx context.Context, user UserContext) (map[string]any, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	holdings, err := s.repo.GetHoldings(ctx, defaultTenant(user.TenantID), user.UserID)
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

func defaultTenant(value string) string {
	if value == "" {
		return "default-tenant"
	}
	return value
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
