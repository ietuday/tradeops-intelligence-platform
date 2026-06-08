package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
)

type RuleConfig struct {
	LargeOrderThreshold          float64
	RapidOrderLimit              int
	RapidOrderWindow             time.Duration
	HighCancelLimit              int
	HighCancelWindow             time.Duration
	RiskScoreThreshold           float64
	AbnormalPriceMovementPercent float64
}

type RuleEngine struct {
	cfg              RuleConfig
	mu               sync.Mutex
	orderTimestamps  map[string][]time.Time
	cancelTimestamps map[string][]time.Time
	lastPrices       map[string]float64
}

func NewRuleEngine(cfg RuleConfig) *RuleEngine {
	return &RuleEngine{
		cfg:              cfg,
		orderTimestamps:  map[string][]time.Time{},
		cancelTimestamps: map[string][]time.Time{},
		lastPrices:       map[string]float64{},
	}
}

func (e *RuleEngine) Evaluate(event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution) {
	switch event.Topic {
	case "order.created", "order.filled":
		return e.evaluateOrder(event, now, event.Topic == "order.created")
	case "order.cancelled":
		return e.evaluateCancelled(event, now)
	case "risk.score.updated":
		return e.evaluateRiskScore(event, now)
	case "market.ticks":
		return e.evaluateMarketTick(event, now)
	default:
		return nil, nil
	}
}

func (e *RuleEngine) evaluateOrder(event domain.SourceEvent, now time.Time, includeRapidRule bool) ([]domain.Alert, []domain.RuleExecution) {
	start := time.Now()
	payload := decodeMap(event.Value)
	entityID := firstString(payload, "orderId", "order_id", "id")
	userID := firstString(payload, "userId", "user_id")
	symbol := strings.ToUpper(firstString(payload, "symbol"))
	price := firstFloat(payload, "price", "limitPrice", "limit_price", "fillPrice", "fill_price")
	quantity := firstFloat(payload, "quantity", "qty")
	notional := firstFloat(payload, "notional", "notionalValue", "notional_value")
	if notional == 0 {
		notional = price * quantity
	}
	var alerts []domain.Alert
	var executions []domain.RuleExecution
	matched := notional > e.cfg.LargeOrderThreshold
	if matched {
		alerts = append(alerts, newAlert("LargeOrderRule", domain.SeverityHigh, "ORDER", entityID, userID, symbol, fmt.Sprintf("Order notional %.2f exceeds configured threshold %.2f", notional, e.cfg.LargeOrderThreshold), map[string]any{"notional": notional, "threshold": e.cfg.LargeOrderThreshold, "sourceTopic": event.Topic}, now))
	}
	executions = append(executions, ruleExecution("LargeOrderRule", event.Topic, entityID, matched, start, nil))

	if includeRapidRule && userID != "" {
		start = time.Now()
		count := e.appendRolling("order", userID, now, e.cfg.RapidOrderWindow)
		matched = count > e.cfg.RapidOrderLimit
		if matched {
			alerts = append(alerts, newAlert("RapidOrderSubmissionRule", domain.SeverityMedium, "USER", userID, userID, symbol, fmt.Sprintf("User submitted %d orders within %s", count, e.cfg.RapidOrderWindow), map[string]any{"count": count, "limit": e.cfg.RapidOrderLimit, "windowSeconds": int(e.cfg.RapidOrderWindow.Seconds()), "sourceTopic": event.Topic}, now))
		}
		executions = append(executions, ruleExecution("RapidOrderSubmissionRule", event.Topic, userID, matched, start, nil))
	}
	return alerts, executions
}

func (e *RuleEngine) evaluateCancelled(event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution) {
	start := time.Now()
	payload := decodeMap(event.Value)
	entityID := firstString(payload, "orderId", "order_id", "id")
	userID := firstString(payload, "userId", "user_id")
	symbol := strings.ToUpper(firstString(payload, "symbol"))
	count := 0
	if userID != "" {
		count = e.appendRolling("cancel", userID, now, e.cfg.HighCancelWindow)
	}
	matched := userID != "" && count > e.cfg.HighCancelLimit
	var alerts []domain.Alert
	if matched {
		alerts = append(alerts, newAlert("HighCancelRateRule", domain.SeverityHigh, "USER", userID, userID, symbol, fmt.Sprintf("User cancelled %d orders within %s", count, e.cfg.HighCancelWindow), map[string]any{"count": count, "limit": e.cfg.HighCancelLimit, "windowSeconds": int(e.cfg.HighCancelWindow.Seconds()), "sourceTopic": event.Topic}, now))
	}
	return alerts, []domain.RuleExecution{ruleExecution("HighCancelRateRule", event.Topic, entityID, matched, start, nil)}
}

func (e *RuleEngine) evaluateRiskScore(event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution) {
	start := time.Now()
	payload := decodeMap(event.Value)
	entityID := firstString(payload, "portfolioId", "portfolio_id", "userId", "user_id", "id")
	userID := firstString(payload, "userId", "user_id")
	score := firstFloat(payload, "score", "riskScore", "risk_score")
	matched := score >= e.cfg.RiskScoreThreshold
	var alerts []domain.Alert
	if matched {
		alerts = append(alerts, newAlert("RiskScoreBreachRule", domain.SeverityCritical, "RISK_SCORE", entityID, userID, "", fmt.Sprintf("Risk score %.2f meets or exceeds threshold %.2f", score, e.cfg.RiskScoreThreshold), map[string]any{"score": score, "threshold": e.cfg.RiskScoreThreshold, "sourceTopic": event.Topic}, now))
	}
	return alerts, []domain.RuleExecution{ruleExecution("RiskScoreBreachRule", event.Topic, entityID, matched, start, nil)}
}

func (e *RuleEngine) evaluateMarketTick(event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution) {
	start := time.Now()
	payload := decodeMap(event.Value)
	symbol := strings.ToUpper(firstString(payload, "symbol"))
	price := firstFloat(payload, "price", "lastPrice", "last_price")
	matched := false
	metadata := map[string]any{"currentPrice": price, "thresholdPercent": e.cfg.AbnormalPriceMovementPercent, "sourceTopic": event.Topic}
	if symbol != "" && price > 0 {
		e.mu.Lock()
		last := e.lastPrices[symbol]
		if last > 0 {
			change := math.Abs(price-last) / last * 100
			metadata["lastPrice"] = last
			metadata["changePercent"] = change
			matched = change > e.cfg.AbnormalPriceMovementPercent
		}
		e.lastPrices[symbol] = price
		e.mu.Unlock()
	}
	var alerts []domain.Alert
	if matched {
		alerts = append(alerts, newAlert("AbnormalPriceMovementRule", domain.SeverityMedium, "SYMBOL", symbol, "", symbol, fmt.Sprintf("%s moved more than %.2f%% from the last seen price", symbol, e.cfg.AbnormalPriceMovementPercent), metadata, now))
	}
	return alerts, []domain.RuleExecution{ruleExecution("AbnormalPriceMovementRule", event.Topic, symbol, matched, start, nil)}
}

func (e *RuleEngine) appendRolling(kind, userID string, now time.Time, window time.Duration) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	store := e.orderTimestamps
	if kind == "cancel" {
		store = e.cancelTimestamps
	}
	cutoff := now.Add(-window)
	var kept []time.Time
	for _, ts := range store[userID] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, now)
	store[userID] = kept
	return len(kept)
}

func newAlert(alertType, severity, entityType, entityID, userID, symbol, description string, metadata map[string]any, now time.Time) domain.Alert {
	var userPtr *string
	if userID != "" {
		userPtr = &userID
	}
	var symbolPtr *string
	if symbol != "" {
		symbolPtr = &symbol
	}
	if entityID == "" {
		entityID = uuid.NewString()
	}
	return domain.Alert{
		ID:          uuid.NewString(),
		AlertType:   alertType,
		Severity:    severity,
		EntityType:  entityType,
		EntityID:    entityID,
		UserID:      userPtr,
		Symbol:      symbolPtr,
		Description: description,
		Status:      domain.StatusOpen,
		Metadata:    metadata,
		CreatedAt:   now,
	}
}

func ruleExecution(ruleName, topic, entityID string, matched bool, start time.Time, err error) domain.RuleExecution {
	var entity *string
	if entityID != "" {
		entity = &entityID
	}
	var errorMessage *string
	if err != nil {
		msg := err.Error()
		errorMessage = &msg
	}
	return domain.RuleExecution{
		ID:              uuid.NewString(),
		RuleName:        ruleName,
		SourceTopic:     topic,
		EntityID:        entity,
		Matched:         matched,
		ExecutionTimeMS: time.Since(start).Milliseconds(),
		ErrorMessage:    errorMessage,
	}
}

func decodeMap(data []byte) map[string]any {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
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

func firstFloat(payload map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			switch typed := value.(type) {
			case float64:
				return typed
			case int:
				return float64(typed)
			case json.Number:
				parsed, _ := typed.Float64()
				return parsed
			}
		}
	}
	return 0
}
