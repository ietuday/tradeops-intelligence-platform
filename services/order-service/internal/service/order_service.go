package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/repository"
)

var ErrForbidden = errors.New("forbidden")
var ErrIdempotencyConflict = errors.New("idempotency conflict")
var ErrOrderNotCancellable = errors.New("order cannot be cancelled")
var ErrNotFound = errors.New("not found")

type UserContext struct {
	UserID   string
	TenantID string
	Roles    []string
}

type OrderService struct {
	repo     *repository.OrderRepository
	producer *kafka.Producer
	metrics  *observability.Metrics
}

func NewOrderService(repo *repository.OrderRepository, producer *kafka.Producer, metrics *observability.Metrics) *OrderService {
	return &OrderService{repo: repo, producer: producer, metrics: metrics}
}

func (s *OrderService) CreateOrder(ctx context.Context, user UserContext, idempotencyKey string, requestBody []byte, correlationID string) (domain.Order, bool, error) {
	start := time.Now()
	defer s.metrics.ObserveProcessing(start)

	if !hasAnyRole(user.Roles, "trader", "trading_admin") {
		return domain.Order{}, false, ErrForbidden
	}
	requestHash := hashRequest(requestBody)
	if user.TenantID == "" {
		user.TenantID = "default-tenant"
	}
	if record, err := s.repo.FindIdempotency(ctx, user.TenantID, user.UserID, idempotencyKey); err == nil {
		if record.RequestHash != requestHash {
			return domain.Order{}, false, ErrIdempotencyConflict
		}
		s.metrics.IdempotencyReplays.Inc()
		order, err := s.repo.GetOrder(ctx, user.TenantID, record.OrderID)
		return order, true, err
	} else if !errors.Is(err, repository.ErrNotFound) {
		return domain.Order{}, false, err
	}

	var req domain.CreateOrderRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return domain.Order{}, false, err
	}

	order, events := s.buildOrder(user.UserID, user.TenantID, req, correlationID)
	created, err := s.repo.CreateOrder(ctx, order, events, idempotencyKey, requestHash)
	if err != nil {
		return domain.Order{}, false, err
	}
	s.recordMetrics(created.Status)
	for _, event := range events {
		event.OrderID = created.ID
		if err := s.producer.Publish(ctx, event); err != nil {
			s.metrics.KafkaPublishErrors.Inc()
		}
	}
	return created, false, nil
}

func (s *OrderService) ListOrders(ctx context.Context, user UserContext) ([]domain.Order, error) {
	if !hasAnyRole(user.Roles, "trader", "trading_admin", "risk_manager", "analyst", "viewer") {
		return nil, ErrForbidden
	}
	return s.repo.ListOrders(ctx, user.TenantID, user.UserID, hasAnyRole(user.Roles, "trading_admin", "risk_manager", "analyst", "viewer"))
}

func (s *OrderService) GetOrder(ctx context.Context, user UserContext, id string) (domain.Order, error) {
	if !hasAnyRole(user.Roles, "trader", "trading_admin", "risk_manager", "analyst", "viewer") {
		return domain.Order{}, ErrForbidden
	}
	order, err := s.repo.GetOrder(ctx, user.TenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Order{}, ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}
	if order.UserID != user.UserID && !hasAnyRole(user.Roles, "trading_admin", "risk_manager", "analyst", "viewer") {
		return domain.Order{}, ErrForbidden
	}
	return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, user UserContext, id, correlationID string) (domain.Order, error) {
	start := time.Now()
	defer s.metrics.ObserveProcessing(start)

	if !hasAnyRole(user.Roles, "trader", "trading_admin") {
		return domain.Order{}, ErrForbidden
	}
	existing, err := s.repo.GetOrder(ctx, user.TenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Order{}, ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}
	if existing.UserID != user.UserID && !hasAnyRole(user.Roles, "trading_admin") {
		return domain.Order{}, ErrForbidden
	}
	if existing.Status != domain.StatusAccepted {
		return domain.Order{}, ErrOrderNotCancellable
	}
	event := makeEvent(existing, "order.cancelled", domain.StatusCancelled, correlationID)
	cancelled, err := s.repo.CancelOrder(ctx, user.TenantID, id, correlationID, event)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Order{}, ErrOrderNotCancellable
	}
	if err != nil {
		return domain.Order{}, err
	}
	event.OrderID = cancelled.ID
	if err := s.producer.Publish(ctx, event); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
	}
	s.metrics.OrdersCancelled.Inc()
	return cancelled, nil
}

func (s *OrderService) buildOrder(userID, tenantID string, req domain.CreateOrderRequest, correlationID string) (domain.Order, []domain.OrderEvent) {
	now := time.Now().UTC()
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	order := domain.Order{
		UserID:        userID,
		TenantID:      tenantID,
		Symbol:        strings.ToUpper(strings.TrimSpace(req.Symbol)),
		Side:          strings.ToUpper(strings.TrimSpace(req.Side)),
		OrderType:     strings.ToUpper(strings.TrimSpace(req.OrderType)),
		Quantity:      req.Quantity,
		LimitPrice:    req.LimitPrice,
		StopPrice:     req.StopPrice,
		Status:        domain.StatusCreated,
		CorrelationID: correlationID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	events := []domain.OrderEvent{
		makeEvent(order, "order.created", domain.StatusCreated, correlationID),
	}
	if reason := validateOrder(order); reason != "" {
		order.Status = domain.StatusRejected
		order.RejectReason = &reason
		events = append(events, makeEvent(order, "order.rejected", domain.StatusRejected, correlationID))
		return order, events
	}
	events = append(events, makeEvent(order, "order.validated", domain.StatusValidated, correlationID))
	order.Status = domain.StatusAccepted
	events = append(events, makeEvent(order, "order.accepted", domain.StatusAccepted, correlationID))
	if order.OrderType == domain.OrderTypeMarket {
		fillPrice := simulatedFillPrice(order)
		order.FillPrice = &fillPrice
		order.Status = domain.StatusFilled
		order.FilledAt = &now
		events = append(events, makeEvent(order, "order.filled", domain.StatusFilled, correlationID))
	}
	return order, events
}

func (s *OrderService) recordMetrics(status string) {
	s.metrics.OrdersCreated.Inc()
	switch status {
	case domain.StatusAccepted:
		s.metrics.OrdersAccepted.Inc()
	case domain.StatusFilled:
		s.metrics.OrdersAccepted.Inc()
		s.metrics.OrdersFilled.Inc()
	case domain.StatusRejected:
		s.metrics.OrdersRejected.Inc()
	}
}

func validateOrder(order domain.Order) string {
	if order.Symbol == "" {
		return "symbol is required"
	}
	if order.Side != domain.SideBuy && order.Side != domain.SideSell {
		return "side must be BUY or SELL"
	}
	if order.OrderType != domain.OrderTypeMarket && order.OrderType != domain.OrderTypeLimit && order.OrderType != domain.OrderTypeStopLoss {
		return "orderType must be MARKET, LIMIT, or STOP_LOSS"
	}
	if order.Quantity <= 0 {
		return "quantity must be greater than zero"
	}
	if order.OrderType == domain.OrderTypeLimit && (order.LimitPrice == nil || *order.LimitPrice <= 0) {
		return "LIMIT order requires limitPrice greater than zero"
	}
	if order.OrderType == domain.OrderTypeStopLoss && (order.StopPrice == nil || *order.StopPrice <= 0) {
		return "STOP_LOSS order requires stopPrice greater than zero"
	}
	return ""
}

func makeEvent(order domain.Order, eventType, status, correlationID string) domain.OrderEvent {
	tenantID := order.TenantID
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	return domain.OrderEvent{
		EventID:       uuid.NewString(),
		EventType:     eventType,
		TenantID:      tenantID,
		OrderID:       order.ID,
		UserID:        order.UserID,
		Symbol:        order.Symbol,
		Side:          order.Side,
		OrderType:     order.OrderType,
		Quantity:      order.Quantity,
		Status:        status,
		FillPrice:     order.FillPrice,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
	}
}

func simulatedFillPrice(order domain.Order) float64 {
	if order.LimitPrice != nil && *order.LimitPrice > 0 {
		return *order.LimitPrice
	}
	if order.StopPrice != nil && *order.StopPrice > 0 {
		return *order.StopPrice
	}
	return 100 + float64(len(order.Symbol))
}

func hashRequest(requestBody []byte) string {
	sum := sha256.Sum256(requestBody)
	return hex.EncodeToString(sum[:])
}

func hasAnyRole(roles []string, allowed ...string) bool {
	for _, role := range roles {
		for _, allow := range allowed {
			if role == allow {
				return true
			}
		}
	}
	return false
}
