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

type effectiveRuleSetting struct {
	Enabled          bool
	Severity         string
	ThresholdNumeric float64
	ThresholdCount   int
	ThresholdPercent float64
	Window           time.Duration
}

type RuleEngine struct {
	cfg              RuleConfig
	mu               sync.Mutex
	orderTimestamps  map[string][]time.Time
	cancelTimestamps map[string][]time.Time
	lastPrices       map[string]float64
	ruleConfigs      map[string]map[string]domain.RuleConfig
}

func NewRuleEngine(cfg RuleConfig) *RuleEngine {
	return &RuleEngine{
		cfg:              cfg,
		orderTimestamps:  map[string][]time.Time{},
		cancelTimestamps: map[string][]time.Time{},
		lastPrices:       map[string]float64{},
		ruleConfigs:      map[string]map[string]domain.RuleConfig{},
	}
}

func DefaultDomainRuleConfigs(tenantID string, cfg RuleConfig) []domain.RuleConfig {
	largeDescription := "Triggers when order notional exceeds configured threshold"
	rapidDescription := "Triggers when a user submits too many orders within a rolling window"
	cancelDescription := "Triggers when a user cancels too many orders within a rolling window"
	riskDescription := "Triggers when risk score meets or exceeds configured threshold"
	priceDescription := "Triggers when symbol price moves beyond configured percentage"
	rapidWindow := int(cfg.RapidOrderWindow.Seconds())
	cancelWindow := int(cfg.HighCancelWindow.Seconds())
	return []domain.RuleConfig{
		{ID: uuid.NewString(), TenantID: defaultTenant(tenantID), RuleName: domain.RuleLargeOrder, Enabled: true, Severity: domain.SeverityHigh, ThresholdNumeric: &cfg.LargeOrderThreshold, Config: map[string]any{}, Description: &largeDescription},
		{ID: uuid.NewString(), TenantID: defaultTenant(tenantID), RuleName: domain.RuleRapidOrderSubmission, Enabled: true, Severity: domain.SeverityMedium, ThresholdCount: &cfg.RapidOrderLimit, WindowSeconds: &rapidWindow, Config: map[string]any{}, Description: &rapidDescription},
		{ID: uuid.NewString(), TenantID: defaultTenant(tenantID), RuleName: domain.RuleHighCancelRate, Enabled: true, Severity: domain.SeverityHigh, ThresholdCount: &cfg.HighCancelLimit, WindowSeconds: &cancelWindow, Config: map[string]any{}, Description: &cancelDescription},
		{ID: uuid.NewString(), TenantID: defaultTenant(tenantID), RuleName: domain.RuleRiskScoreBreach, Enabled: true, Severity: domain.SeverityHigh, ThresholdNumeric: &cfg.RiskScoreThreshold, Config: map[string]any{}, Description: &riskDescription},
		{ID: uuid.NewString(), TenantID: defaultTenant(tenantID), RuleName: domain.RuleAbnormalPriceMovement, Enabled: true, Severity: domain.SeverityMedium, ThresholdPercent: &cfg.AbnormalPriceMovementPercent, Config: map[string]any{}, Description: &priceDescription},
	}
}

func fallbackDomainRuleConfig(tenantID, ruleName string, cfg RuleConfig) domain.RuleConfig {
	for _, config := range DefaultDomainRuleConfigs(tenantID, cfg) {
		if config.RuleName == ruleName {
			return config
		}
	}
	return domain.RuleConfig{}
}

func (e *RuleEngine) ruleSetting(tenantID, ruleName string) effectiveRuleSetting {
	setting := e.fallbackSetting(ruleName)
	e.mu.Lock()
	configs := e.ruleConfigs[defaultTenant(tenantID)]
	if configs == nil && defaultTenant(tenantID) != "default-tenant" {
		configs = e.ruleConfigs["default-tenant"]
	}
	config, ok := configs[ruleName]
	e.mu.Unlock()
	if !ok {
		return setting
	}
	setting.Enabled = config.Enabled
	if config.Severity != "" {
		setting.Severity = config.Severity
	}
	if config.ThresholdNumeric != nil {
		setting.ThresholdNumeric = *config.ThresholdNumeric
	}
	if config.ThresholdCount != nil {
		setting.ThresholdCount = *config.ThresholdCount
	}
	if config.WindowSeconds != nil && *config.WindowSeconds > 0 {
		setting.Window = time.Duration(*config.WindowSeconds) * time.Second
	}
	if config.ThresholdPercent != nil {
		setting.ThresholdPercent = *config.ThresholdPercent
	}
	return setting
}

func (e *RuleEngine) fallbackSetting(ruleName string) effectiveRuleSetting {
	switch ruleName {
	case domain.RuleLargeOrder:
		return effectiveRuleSetting{Enabled: true, Severity: domain.SeverityHigh, ThresholdNumeric: e.cfg.LargeOrderThreshold}
	case domain.RuleRapidOrderSubmission:
		return effectiveRuleSetting{Enabled: true, Severity: domain.SeverityMedium, ThresholdCount: e.cfg.RapidOrderLimit, Window: e.cfg.RapidOrderWindow}
	case domain.RuleHighCancelRate:
		return effectiveRuleSetting{Enabled: true, Severity: domain.SeverityHigh, ThresholdCount: e.cfg.HighCancelLimit, Window: e.cfg.HighCancelWindow}
	case domain.RuleRiskScoreBreach:
		return effectiveRuleSetting{Enabled: true, Severity: domain.SeverityCritical, ThresholdNumeric: e.cfg.RiskScoreThreshold}
	case domain.RuleAbnormalPriceMovement:
		return effectiveRuleSetting{Enabled: true, Severity: domain.SeverityMedium, ThresholdPercent: e.cfg.AbnormalPriceMovementPercent}
	default:
		return effectiveRuleSetting{Enabled: true, Severity: domain.SeverityMedium}
	}
}

func (e *RuleEngine) Evaluate(event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution) {
	alerts, executions, _ := e.EvaluateForTenant("default-tenant", event, now)
	return alerts, executions
}

func (e *RuleEngine) EvaluateForTenant(tenantID string, event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution, []string) {
	switch event.Topic {
	case "order.created", "order.filled":
		alerts, executions, skipped := e.evaluateOrder(tenantID, event, now, event.Topic == "order.created")
		return alerts, executions, skipped
	case "order.cancelled":
		alerts, executions, skipped := e.evaluateCancelled(tenantID, event, now)
		return alerts, executions, skipped
	case "risk.score.updated":
		alerts, executions, skipped := e.evaluateRiskScore(tenantID, event, now)
		return alerts, executions, skipped
	case "market.ticks":
		alerts, executions, skipped := e.evaluateMarketTick(tenantID, event, now)
		return alerts, executions, skipped
	default:
		return nil, nil, nil
	}
}

func (e *RuleEngine) SetTenantRuleConfigs(tenantID string, configs []domain.RuleConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.ruleConfigs == nil {
		e.ruleConfigs = map[string]map[string]domain.RuleConfig{}
	}
	byRule := map[string]domain.RuleConfig{}
	for _, config := range configs {
		byRule[config.RuleName] = config
	}
	e.ruleConfigs[defaultTenant(tenantID)] = byRule
}

func (e *RuleEngine) evaluateOrder(tenantID string, event domain.SourceEvent, now time.Time, includeRapidRule bool) ([]domain.Alert, []domain.RuleExecution, []string) {
	start := time.Now()
	largeRule := e.ruleSetting(tenantID, domain.RuleLargeOrder)
	rapidRule := e.ruleSetting(tenantID, domain.RuleRapidOrderSubmission)
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
	var skipped []string
	if !largeRule.Enabled {
		skipped = append(skipped, domain.RuleLargeOrder)
	} else {
		matched := notional > largeRule.ThresholdNumeric
		if matched {
			alerts = append(alerts, newAlert(domain.RuleLargeOrder, largeRule.Severity, "ORDER", entityID, userID, symbol, fmt.Sprintf("Order notional %.2f exceeds configured threshold %.2f", notional, largeRule.ThresholdNumeric), map[string]any{"notional": notional, "threshold": largeRule.ThresholdNumeric, "sourceTopic": event.Topic}, now))
		}
		executions = append(executions, ruleExecution(domain.RuleLargeOrder, event.Topic, entityID, matched, start, nil))
	}

	if includeRapidRule && userID != "" {
		start = time.Now()
		if !rapidRule.Enabled {
			skipped = append(skipped, domain.RuleRapidOrderSubmission)
		} else {
			count := e.appendRolling("order", userID, now, rapidRule.Window)
			matched := count > rapidRule.ThresholdCount
			if matched {
				alerts = append(alerts, newAlert(domain.RuleRapidOrderSubmission, rapidRule.Severity, "USER", userID, userID, symbol, fmt.Sprintf("User submitted %d orders within %s", count, rapidRule.Window), map[string]any{"count": count, "limit": rapidRule.ThresholdCount, "windowSeconds": int(rapidRule.Window.Seconds()), "sourceTopic": event.Topic}, now))
			}
			executions = append(executions, ruleExecution(domain.RuleRapidOrderSubmission, event.Topic, userID, matched, start, nil))
		}
	}
	return alerts, executions, skipped
}

func (e *RuleEngine) evaluateCancelled(tenantID string, event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution, []string) {
	start := time.Now()
	rule := e.ruleSetting(tenantID, domain.RuleHighCancelRate)
	if !rule.Enabled {
		return nil, nil, []string{domain.RuleHighCancelRate}
	}
	payload := decodeMap(event.Value)
	entityID := firstString(payload, "orderId", "order_id", "id")
	userID := firstString(payload, "userId", "user_id")
	symbol := strings.ToUpper(firstString(payload, "symbol"))
	count := 0
	if userID != "" {
		count = e.appendRolling("cancel", userID, now, rule.Window)
	}
	matched := userID != "" && count > rule.ThresholdCount
	var alerts []domain.Alert
	if matched {
		alerts = append(alerts, newAlert(domain.RuleHighCancelRate, rule.Severity, "USER", userID, userID, symbol, fmt.Sprintf("User cancelled %d orders within %s", count, rule.Window), map[string]any{"count": count, "limit": rule.ThresholdCount, "windowSeconds": int(rule.Window.Seconds()), "sourceTopic": event.Topic}, now))
	}
	return alerts, []domain.RuleExecution{ruleExecution(domain.RuleHighCancelRate, event.Topic, entityID, matched, start, nil)}, nil
}

func (e *RuleEngine) evaluateRiskScore(tenantID string, event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution, []string) {
	start := time.Now()
	rule := e.ruleSetting(tenantID, domain.RuleRiskScoreBreach)
	if !rule.Enabled {
		return nil, nil, []string{domain.RuleRiskScoreBreach}
	}
	payload := decodeMap(event.Value)
	entityID := firstString(payload, "portfolioId", "portfolio_id", "userId", "user_id", "id")
	userID := firstString(payload, "userId", "user_id")
	score := firstFloat(payload, "score", "riskScore", "risk_score")
	matched := score >= rule.ThresholdNumeric
	var alerts []domain.Alert
	if matched {
		alerts = append(alerts, newAlert(domain.RuleRiskScoreBreach, rule.Severity, "RISK_SCORE", entityID, userID, "", fmt.Sprintf("Risk score %.2f meets or exceeds threshold %.2f", score, rule.ThresholdNumeric), map[string]any{"score": score, "threshold": rule.ThresholdNumeric, "sourceTopic": event.Topic}, now))
	}
	return alerts, []domain.RuleExecution{ruleExecution(domain.RuleRiskScoreBreach, event.Topic, entityID, matched, start, nil)}, nil
}

func (e *RuleEngine) evaluateMarketTick(tenantID string, event domain.SourceEvent, now time.Time) ([]domain.Alert, []domain.RuleExecution, []string) {
	start := time.Now()
	rule := e.ruleSetting(tenantID, domain.RuleAbnormalPriceMovement)
	if !rule.Enabled {
		return nil, nil, []string{domain.RuleAbnormalPriceMovement}
	}
	payload := decodeMap(event.Value)
	symbol := strings.ToUpper(firstString(payload, "symbol"))
	price := firstFloat(payload, "price", "lastPrice", "last_price")
	matched := false
	metadata := map[string]any{"currentPrice": price, "thresholdPercent": rule.ThresholdPercent, "sourceTopic": event.Topic}
	if symbol != "" && price > 0 {
		e.mu.Lock()
		last := e.lastPrices[symbol]
		if last > 0 {
			change := math.Abs(price-last) / last * 100
			metadata["lastPrice"] = last
			metadata["changePercent"] = change
			matched = change > rule.ThresholdPercent
		}
		e.lastPrices[symbol] = price
		e.mu.Unlock()
	}
	var alerts []domain.Alert
	if matched {
		alerts = append(alerts, newAlert(domain.RuleAbnormalPriceMovement, rule.Severity, "SYMBOL", symbol, "", symbol, fmt.Sprintf("%s moved more than %.2f%% from the last seen price", symbol, rule.ThresholdPercent), metadata, now))
	}
	return alerts, []domain.RuleExecution{ruleExecution(domain.RuleAbnormalPriceMovement, event.Topic, symbol, matched, start, nil)}, nil
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
