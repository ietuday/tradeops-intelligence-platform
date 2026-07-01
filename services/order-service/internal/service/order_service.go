package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/expiry"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/risk"
)

var ErrForbidden = errors.New("forbidden")
var ErrIdempotencyConflict = errors.New("idempotency conflict")
var ErrOrderNotCancellable = errors.New("order cannot be cancelled")
var ErrInvalidOrder = errors.New("invalid order")
var ErrVersionConflict = errors.New("stale order version")
var ErrNotFound = errors.New("not found")
var ErrPreTradeRiskRejected = errors.New("pre-trade risk rejected")
var ErrPreTradeRiskUnavailable = errors.New("pre-trade risk unavailable")
var ErrPreTradeRiskInvalidResponse = errors.New("pre-trade risk invalid response")

type UserContext struct {
	UserID   string
	TenantID string
	Roles    []string
}

type OrderService struct {
	repo     *repository.OrderRepository
	metrics  *observability.Metrics
	calendar expiry.TradingCalendar
	risk     risk.PreTradeRiskChecker
	riskCfg  RiskOptions
}

type RiskOptions struct {
	Enabled  bool
	FailOpen bool
}

func NewOrderService(repo *repository.OrderRepository, producer *kafka.Producer, metrics *observability.Metrics, calendar expiry.TradingCalendar, opts ...RiskOptions) *OrderService {
	_ = producer
	riskCfg := RiskOptions{}
	if len(opts) > 0 {
		riskCfg = opts[0]
	}
	return &OrderService{repo: repo, metrics: metrics, calendar: calendar, riskCfg: riskCfg}
}

func (s *OrderService) SetRiskChecker(checker risk.PreTradeRiskChecker) {
	s.risk = checker
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
		if err != nil {
			return order, true, err
		}
		if order.Status == domain.StatusRiskRejected {
			return order, true, fmt.Errorf("%w: %s", ErrPreTradeRiskRejected, stringValue(order.RiskReasonMessage, "Risk rejected order"))
		}
		if order.Status == domain.StatusRiskError {
			if stringValue(order.RiskReasonCode, "") == risk.ReasonResponseInvalid {
				return order, true, fmt.Errorf("%w: %s", ErrPreTradeRiskInvalidResponse, stringValue(order.RiskReasonMessage, "Risk response invalid"))
			}
			return order, true, fmt.Errorf("%w: %s", ErrPreTradeRiskUnavailable, stringValue(order.RiskReasonMessage, "Risk Engine unavailable"))
		}
		return order, true, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return domain.Order{}, false, err
	}

	var req domain.CreateOrderRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return domain.Order{}, false, err
	}

	order, events, err := s.buildOrder(ctx, user.UserID, user.TenantID, req, correlationID)
	if err != nil {
		return domain.Order{}, false, err
	}
	created, _, err := s.repo.CreateOrder(ctx, order, events, idempotencyKey, requestHash)
	if err != nil {
		return domain.Order{}, false, err
	}
	if created.Status == domain.StatusRiskPending {
		finalized, err := s.evaluateAndApplyRisk(ctx, created)
		if err != nil {
			return finalized, false, err
		}
		created = finalized
	}
	s.recordMetrics(created.Status)
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
	s.metrics.OrdersCancelled.Inc()
	return cancelled, nil
}

func (s *OrderService) AmendOrder(ctx context.Context, user UserContext, id string, req domain.AmendOrderRequest, correlationID string) (domain.Order, error) {
	if !hasAnyRole(user.Roles, "trader", "trading_admin") {
		return domain.Order{}, ErrForbidden
	}
	if req.ExpectedVersion <= 0 {
		return domain.Order{}, fmt.Errorf("%w: expectedVersion is required", ErrInvalidOrder)
	}
	existing, err := s.GetOrder(ctx, user, id)
	if err != nil {
		return domain.Order{}, err
	}
	if existing.UserID != user.UserID && !hasAnyRole(user.Roles, "trading_admin") {
		return domain.Order{}, ErrForbidden
	}
	if existing.Status != domain.StatusAccepted && existing.Status != domain.StatusPartiallyFilled {
		return domain.Order{}, ErrOrderNotCancellable
	}
	if req.Quantity != nil && *req.Quantity < existing.FilledQuantity {
		return domain.Order{}, fmt.Errorf("%w: quantity cannot be below filledQuantity", ErrInvalidOrder)
	}
	if req.ExpiresAt != nil {
		if existing.TimeInForce != domain.TimeInForceGTD {
			return domain.Order{}, fmt.Errorf("%w: expiresAt can only be amended for GTD orders", ErrInvalidOrder)
		}
		expiresAt := req.ExpiresAt.UTC()
		if !expiresAt.After(time.Now().UTC()) {
			return domain.Order{}, fmt.Errorf("%w: GTD expiresAt must be in the future", ErrInvalidOrder)
		}
		req.ExpiresAt = &expiresAt
	}
	order, err := s.repo.AmendOrder(ctx, user.TenantID, id, req, correlationID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Order{}, ErrVersionConflict
	}
	if err != nil {
		return domain.Order{}, err
	}
	s.metrics.OrdersAmended.Inc()
	return order, nil
}

func (s *OrderService) ListExecutionsForOrder(ctx context.Context, user UserContext, orderID string) ([]domain.Execution, error) {
	if _, err := s.GetOrder(ctx, user, orderID); err != nil {
		return nil, err
	}
	return s.repo.ListExecutionsForOrder(ctx, user.TenantID, orderID)
}

func (s *OrderService) ListTrades(ctx context.Context, user UserContext, limit int) ([]domain.Execution, error) {
	if !hasAnyRole(user.Roles, "trader", "trading_admin", "risk_manager", "analyst", "viewer") {
		return nil, ErrForbidden
	}
	return s.repo.ListTrades(ctx, user.TenantID, limit)
}

func (s *OrderService) GetTrade(ctx context.Context, user UserContext, id string) (domain.Execution, error) {
	if !hasAnyRole(user.Roles, "trader", "trading_admin", "risk_manager", "analyst", "viewer") {
		return domain.Execution{}, ErrForbidden
	}
	trade, err := s.repo.GetTrade(ctx, user.TenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Execution{}, ErrNotFound
	}
	return trade, err
}

func (s *OrderService) OrderBookDepth(ctx context.Context, user UserContext, symbol string, depth int) (map[string]any, error) {
	if !hasAnyRole(user.Roles, "trader", "trading_admin", "risk_manager", "analyst", "viewer") {
		return nil, ErrForbidden
	}
	return s.repo.OrderBookDepth(ctx, user.TenantID, strings.ToUpper(strings.TrimSpace(symbol)), depth)
}

func (s *OrderService) buildOrder(ctx context.Context, userID, tenantID string, req domain.CreateOrderRequest, correlationID string) (domain.Order, []domain.OrderEvent, error) {
	now := time.Now().UTC()
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	timeInForce := normalizeTimeInForce(req.TimeInForce)
	expiresAt := req.ExpiresAt
	if expiresAt != nil {
		utc := expiresAt.UTC()
		expiresAt = &utc
	}
	order := domain.Order{
		UserID:            userID,
		TenantID:          tenantID,
		Symbol:            strings.ToUpper(strings.TrimSpace(req.Symbol)),
		Side:              strings.ToUpper(strings.TrimSpace(req.Side)),
		OrderType:         normalizeOrderType(req.OrderType),
		Quantity:          req.Quantity,
		FilledQuantity:    0,
		RemainingQuantity: req.Quantity,
		LimitPrice:        req.LimitPrice,
		StopPrice:         req.StopPrice,
		TimeInForce:       timeInForce,
		ExpiresAt:         expiresAt,
		Version:           1,
		Status:            domain.StatusCreated,
		RiskStatus:        "NOT_EVALUATED",
		CorrelationID:     correlationID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	events := []domain.OrderEvent{
		makeEvent(order, "order.created", domain.StatusCreated, correlationID),
	}
	if reason := validateOrder(order); reason != "" {
		order.Status = domain.StatusRejected
		order.RejectReason = &reason
		events = append(events, makeEvent(order, "order.rejected", domain.StatusRejected, correlationID))
		return order, events, nil
	}
	if order.TimeInForce == domain.TimeInForceDay {
		if s.calendar == nil {
			return domain.Order{}, nil, fmt.Errorf("%w: DAY trading calendar is not configured", ErrInvalidOrder)
		}
		dayExpiry, err := s.calendar.ExpiryForDayOrder(ctx, now)
		if err != nil {
			return domain.Order{}, nil, err
		}
		order.ExpiresAt = &dayExpiry
	}
	events = append(events, makeEvent(order, "order.validated", domain.StatusValidated, correlationID))
	if s.riskCfg.Enabled && s.risk != nil {
		order.Status = domain.StatusRiskPending
		order.RiskStatus = "PENDING"
		events = append(events, makeEvent(order, "order.risk_pending", domain.StatusRiskPending, correlationID))
		return order, events, nil
	}
	order.Status = domain.StatusAccepted
	order.RiskStatus = "NOT_EVALUATED"
	events = append(events, makeEvent(order, "order.accepted", domain.StatusAccepted, correlationID))
	return order, events, nil
}

func (s *OrderService) evaluateAndApplyRisk(ctx context.Context, order domain.Order) (domain.Order, error) {
	start := time.Now()
	s.metrics.PreTradeRiskInflight.Inc()
	defer s.metrics.PreTradeRiskInflight.Dec()
	s.metrics.PreTradeRiskRequests.WithLabelValues(order.OrderType, order.Side).Inc()
	req, estimatedPrice, estimatedNotional, reason := s.buildRiskRequest(order)
	if reason != "" {
		decision := risk.PreTradeRiskDecision{
			DecisionID:       uuid.NewString(),
			Decision:         risk.DecisionRejected,
			Approved:         false,
			ReasonCode:       risk.ReasonReferenceUnavailable,
			ReasonMessage:    reason,
			EvaluatedAt:      time.Now().UTC(),
			PolicyVersion:    "pretrade-v1",
			EvaluatedLimits:  map[string]string{},
			RequestSnapshot:  requestSnapshot(req),
			ResponseSnapshot: map[string]any{"decision": risk.DecisionRejected, "reasonCode": risk.ReasonReferenceUnavailable},
		}
		s.observeRiskDecision(start, order, decision, ErrPreTradeRiskRejected)
		return s.persistRiskDecision(ctx, order, decision, estimatedPrice, estimatedNotional, ErrPreTradeRiskRejected)
	}
	decision, err := s.risk.Evaluate(ctx, req)
	if decision.DecisionID == "" {
		decision.DecisionID = uuid.NewString()
	}
	if decision.RequestSnapshot == nil {
		decision.RequestSnapshot = requestSnapshot(req)
	}
	if err != nil && s.riskCfg.FailOpen && (errors.Is(err, risk.ErrUnavailable) || errors.Is(err, risk.ErrTimeout)) {
		decision = risk.PreTradeRiskDecision{
			DecisionID:       uuid.NewString(),
			Approved:         true,
			Decision:         risk.DecisionApproved,
			ReasonCode:       risk.ReasonServiceBypassed,
			ReasonMessage:    "Risk Engine unavailable; fail-open bypass applied",
			EvaluatedLimits:  map[string]string{},
			EvaluatedAt:      time.Now().UTC(),
			PolicyVersion:    "fail-open",
			RequestSnapshot:  requestSnapshot(req),
			ResponseSnapshot: map[string]any{"decision": risk.DecisionApproved, "reasonCode": risk.ReasonServiceBypassed},
		}
		err = nil
	}
	applyErr := error(nil)
	if err != nil {
		switch {
		case errors.Is(err, risk.ErrRejected):
			applyErr = ErrPreTradeRiskRejected
		case errors.Is(err, risk.ErrInvalidResponse):
			applyErr = ErrPreTradeRiskInvalidResponse
		default:
			applyErr = ErrPreTradeRiskUnavailable
		}
	}
	s.observeRiskDecision(start, order, decision, applyErr)
	return s.persistRiskDecision(ctx, order, decision, estimatedPrice, estimatedNotional, applyErr)
}

func (s *OrderService) observeRiskDecision(start time.Time, order domain.Order, decision risk.PreTradeRiskDecision, err error) {
	result := "approved"
	if err != nil || !decision.Approved {
		result = "rejected"
	}
	if errors.Is(err, ErrPreTradeRiskUnavailable) {
		result = "unavailable"
	}
	if errors.Is(err, ErrPreTradeRiskInvalidResponse) {
		result = "invalid_response"
	}
	reasonCode := decision.ReasonCode
	if reasonCode == "" {
		reasonCode = "UNKNOWN"
	}
	s.metrics.PreTradeRiskDuration.WithLabelValues(result, reasonCode, order.OrderType, order.Side).Observe(time.Since(start).Seconds())
	switch {
	case decision.ReasonCode == risk.ReasonServiceBypassed:
		s.metrics.PreTradeRiskBypassed.Inc()
		s.metrics.PreTradeRiskApproved.WithLabelValues(reasonCode, order.OrderType, order.Side).Inc()
	case decision.Approved && err == nil:
		s.metrics.PreTradeRiskApproved.WithLabelValues(reasonCode, order.OrderType, order.Side).Inc()
	case errors.Is(err, ErrPreTradeRiskUnavailable):
		s.metrics.PreTradeRiskErrors.WithLabelValues(result, reasonCode, order.OrderType, order.Side).Inc()
		if decision.ReasonCode == risk.ReasonServiceTimeout {
			s.metrics.PreTradeRiskTimeouts.Inc()
		}
	case errors.Is(err, ErrPreTradeRiskInvalidResponse):
		s.metrics.PreTradeRiskErrors.WithLabelValues(result, reasonCode, order.OrderType, order.Side).Inc()
	default:
		s.metrics.PreTradeRiskRejected.WithLabelValues(reasonCode, order.OrderType, order.Side).Inc()
	}
}

func (s *OrderService) persistRiskDecision(ctx context.Context, order domain.Order, decision risk.PreTradeRiskDecision, estimatedPrice, estimatedNotional float64, resultErr error) (domain.Order, error) {
	domainDecision := domain.RiskDecision{
		DecisionID:        decision.DecisionID,
		TenantID:          order.TenantID,
		OrderID:           order.ID,
		UserID:            order.UserID,
		PolicyID:          decision.PolicyID,
		PolicyVersion:     decision.PolicyVersion,
		Decision:          decision.Decision,
		Approved:          decision.Approved,
		ReasonCode:        decision.ReasonCode,
		ReasonMessage:     decision.ReasonMessage,
		EstimatedPrice:    estimatedPrice,
		EstimatedNotional: estimatedNotional,
		EvaluatedLimits:   decision.EvaluatedLimits,
		RequestSnapshot:   decision.RequestSnapshot,
		ResponseSnapshot:  decision.ResponseSnapshot,
		CorrelationID:     order.CorrelationID,
		EvaluatedAt:       decision.EvaluatedAt,
	}
	accepted := makeEvent(order, "order.accepted", domain.StatusAccepted, order.CorrelationID)
	eventType := "order.risk_rejected"
	status := domain.StatusRiskRejected
	if decision.Decision == risk.DecisionUnavailable || decision.Decision == risk.DecisionInvalidResponse {
		eventType = "order.risk_error"
		status = domain.StatusRiskError
	}
	rejected := makeEvent(order, eventType, status, order.CorrelationID)
	rejected.DecisionID = decision.DecisionID
	rejected.ReasonCode = decision.ReasonCode
	rejected.ReasonMessage = decision.ReasonMessage
	rejected.PolicyVersion = decision.PolicyVersion
	rejected.EvaluatedAt = &decision.EvaluatedAt
	rejected.EstimatedPrice = &estimatedPrice
	rejected.EstimatedNotional = &estimatedNotional
	rejected.Source = "order-service"
	finalized, _, err := s.repo.ApplyRiskDecision(ctx, order.TenantID, order.ID, domainDecision, []domain.OrderEvent{accepted}, rejected)
	if err != nil {
		return domain.Order{}, err
	}
	if resultErr != nil {
		return finalized, fmt.Errorf("%w: %s", resultErr, decision.ReasonMessage)
	}
	return finalized, nil
}

func (s *OrderService) buildRiskRequest(order domain.Order) (risk.PreTradeRiskRequest, float64, float64, string) {
	estimatedPrice := 0.0
	if order.LimitPrice != nil && (order.OrderType == domain.OrderTypeLimit || order.OrderType == domain.OrderTypeStopLimit) {
		estimatedPrice = *order.LimitPrice
	}
	if estimatedPrice <= 0 {
		return risk.PreTradeRiskRequest{}, 0, 0, "Reference price unavailable for pre-trade risk evaluation"
	}
	estimatedNotional := estimatedPrice * order.Quantity
	if estimatedNotional <= 0 {
		return risk.PreTradeRiskRequest{}, estimatedPrice, estimatedNotional, "Estimated notional is invalid"
	}
	req := risk.PreTradeRiskRequest{
		TenantID:          order.TenantID,
		UserID:            order.UserID,
		OrderID:           order.ID,
		Symbol:            order.Symbol,
		Side:              order.Side,
		OrderType:         order.OrderType,
		Quantity:          formatDecimal(order.Quantity),
		EstimatedPrice:    formatDecimal(estimatedPrice),
		EstimatedNotional: formatDecimal(estimatedNotional),
		TimeInForce:       order.TimeInForce,
		Currency:          "USD",
		SubmittedAt:       order.CreatedAt,
		CorrelationID:     order.CorrelationID,
	}
	if order.LimitPrice != nil {
		value := formatDecimal(*order.LimitPrice)
		req.LimitPrice = &value
	}
	if order.StopPrice != nil {
		value := formatDecimal(*order.StopPrice)
		req.StopPrice = &value
	}
	return req, estimatedPrice, estimatedNotional, ""
}

func requestSnapshot(req risk.PreTradeRiskRequest) map[string]any {
	body, _ := json.Marshal(req)
	var snapshot map[string]any
	_ = json.Unmarshal(body, &snapshot)
	return snapshot
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
	case domain.StatusRiskRejected, domain.StatusRiskError:
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
	if order.OrderType != domain.OrderTypeMarket && order.OrderType != domain.OrderTypeLimit && order.OrderType != domain.OrderTypeStop && order.OrderType != domain.OrderTypeStopLimit && order.OrderType != domain.OrderTypeStopLoss {
		return "orderType must be MARKET, LIMIT, STOP, STOP_LIMIT, or STOP_LOSS"
	}
	if order.Quantity <= 0 {
		return "quantity must be greater than zero"
	}
	if order.FilledQuantity < 0 || order.FilledQuantity > order.Quantity {
		return "filledQuantity must be between zero and quantity"
	}
	if (order.RemainingQuantity != 0 || order.FilledQuantity != 0) && order.RemainingQuantity != order.Quantity-order.FilledQuantity {
		return "remainingQuantity must equal quantity minus filledQuantity"
	}
	if order.OrderType == domain.OrderTypeLimit && (order.LimitPrice == nil || *order.LimitPrice <= 0) {
		return "LIMIT order requires limitPrice greater than zero"
	}
	if (order.OrderType == domain.OrderTypeStop || order.OrderType == domain.OrderTypeStopLoss) && (order.StopPrice == nil || *order.StopPrice <= 0) {
		return "STOP order requires stopPrice greater than zero"
	}
	if order.OrderType == domain.OrderTypeStopLimit {
		if order.StopPrice == nil || *order.StopPrice <= 0 {
			return "STOP_LIMIT order requires stopPrice greater than zero"
		}
		if order.LimitPrice == nil || *order.LimitPrice <= 0 {
			return "STOP_LIMIT order requires limitPrice greater than zero"
		}
	}
	if order.TimeInForce != "" && order.TimeInForce != domain.TimeInForceDay && order.TimeInForce != domain.TimeInForceGTC && order.TimeInForce != domain.TimeInForceGTD && order.TimeInForce != domain.TimeInForceIOC && order.TimeInForce != domain.TimeInForceFOK {
		return "timeInForce must be DAY, GTC, GTD, IOC, or FOK"
	}
	if order.TimeInForce == domain.TimeInForceGTD {
		if order.ExpiresAt == nil || !order.ExpiresAt.After(time.Now().UTC()) {
			return "GTD orders require future expiresAt"
		}
	}
	if order.TimeInForce != domain.TimeInForceGTD && order.ExpiresAt != nil {
		return "expiresAt is only valid for GTD orders"
	}
	return ""
}

func makeEvent(order domain.Order, eventType, status, correlationID string) domain.OrderEvent {
	tenantID := order.TenantID
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	event := domain.OrderEvent{
		EventID:           uuid.NewString(),
		EventType:         eventType,
		TenantID:          tenantID,
		OrderID:           order.ID,
		UserID:            order.UserID,
		Symbol:            order.Symbol,
		Side:              order.Side,
		OrderType:         order.OrderType,
		TimeInForce:       order.TimeInForce,
		Quantity:          order.Quantity,
		FilledQuantity:    order.FilledQuantity,
		RemainingQuantity: order.RemainingQuantity,
		Status:            status,
		FillPrice:         order.FillPrice,
		AverageFillPrice:  order.AverageFillPrice,
		Version:           order.Version,
		ExpiresAt:         order.ExpiresAt,
		ExpiredAt:         order.ExpiredAt,
		OccurredAt:        time.Now().UTC(),
		CorrelationID:     correlationID,
	}
	if order.ExpiryReason != nil {
		event.ExpiryReason = *order.ExpiryReason
	}
	return event
}

func normalizeTimeInForce(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return domain.TimeInForceDay
	}
	return value
}

func normalizeOrderType(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == domain.OrderTypeStopLoss {
		return domain.OrderTypeStop
	}
	return value
}

func formatDecimal(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
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

func stringValue(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}
