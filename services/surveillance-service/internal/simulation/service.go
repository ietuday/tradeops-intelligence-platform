package simulation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	ruleservice "github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/service"
)

var ErrInvalidRequest = errors.New("invalid rule simulation request")

type RuleConfigProvider interface {
	LoadEffectiveRuleConfigs(ctx context.Context, tenantID string) ([]domain.RuleConfig, error)
}

type EventPublisher interface {
	PublishRuleSimulation(ctx context.Context, event domain.RuleSimulationEvent) error
}

type Service struct {
	configs   RuleConfigProvider
	publisher EventPublisher
	metrics   *observability.Metrics
	fallback  ruleservice.RuleConfig
	loader    HistoricalEventLoader
}

func NewService(configs RuleConfigProvider, publisher EventPublisher, metrics *observability.Metrics, fallback ruleservice.RuleConfig, loader HistoricalEventLoader) *Service {
	if loader == nil {
		loader = NewDemoEventLoader()
	}
	if metrics == nil {
		metrics = observability.NewMetrics()
	}
	return &Service{configs: configs, publisher: publisher, metrics: metrics, fallback: fallback, loader: loader}
}

func (s *Service) Simulate(ctx context.Context, request RuleSimulationRequest, correlationID string) (RuleSimulationResult, error) {
	lookback := lookbackMinutes(request.LookbackMinutes)
	ruleName, err := normalizeRuleName(request.RuleName)
	if err != nil {
		s.recordFailure(ctx, request.TenantID, request.RuleName, lookback, correlationID, err)
		return RuleSimulationResult{}, err
	}
	request.RuleName = ruleName
	if err := validateRequest(request, ruleName); err != nil {
		s.recordFailure(ctx, request.TenantID, ruleName, lookback, correlationID, err)
		return RuleSimulationResult{}, err
	}
	s.publish(ctx, "surveillance.rule_simulation.requested", request.TenantID, ruleName, lookback, 0, 0, "requested", "", correlationID)
	start := time.Now()
	result, err := s.run(ctx, request.TenantID, ruleName, request.Config, lookback)
	status := "completed"
	if err != nil {
		status = "failed"
		s.metrics.RuleSimulationDuration.WithLabelValues(ruleName, status).Observe(time.Since(start).Seconds())
		s.recordFailure(ctx, request.TenantID, ruleName, lookback, correlationID, err)
		return RuleSimulationResult{}, err
	}
	s.metrics.RuleSimulationRequests.WithLabelValues(ruleName, status).Inc()
	s.metrics.RuleSimulationDuration.WithLabelValues(ruleName, status).Observe(time.Since(start).Seconds())
	s.metrics.RuleSimulationMatches.WithLabelValues(ruleName).Add(float64(result.MatchedEvents))
	s.publish(ctx, "surveillance.rule_simulation.completed", request.TenantID, ruleName, lookback, result.MatchedEvents, result.WouldTriggerAlerts, status, "", correlationID)
	return result, nil
}

func (s *Service) SimulateBulk(ctx context.Context, request RuleSimulationRequest, correlationID string) (BulkSimulationResponse, error) {
	lookback := lookbackMinutes(request.LookbackMinutes)
	if strings.TrimSpace(request.TenantID) == "" {
		return BulkSimulationResponse{}, fmt.Errorf("%w: tenantId is required", ErrInvalidRequest)
	}
	if dryRun := request.DryRun; dryRun != nil && !*dryRun {
		return BulkSimulationResponse{}, fmt.Errorf("%w: simulation endpoints only support dryRun=true", ErrInvalidRequest)
	}
	items := request.Rules
	if len(items) == 0 && strings.TrimSpace(request.RuleName) != "" {
		items = []RuleSimulation{{RuleName: request.RuleName, Config: request.Config}}
	}
	if len(items) == 0 {
		return BulkSimulationResponse{}, fmt.Errorf("%w: at least one rule is required", ErrInvalidRequest)
	}
	response := BulkSimulationResponse{TenantID: request.TenantID, DryRun: true, LookbackMinutes: lookback}
	for _, item := range items {
		result, err := s.Simulate(ctx, RuleSimulationRequest{
			TenantID:        request.TenantID,
			RuleName:        item.RuleName,
			Config:          item.Config,
			LookbackMinutes: &lookback,
			DryRun:          boolPtr(true),
		}, correlationID)
		if err != nil {
			return BulkSimulationResponse{}, err
		}
		response.Results = append(response.Results, result)
	}
	return response, nil
}

func (s *Service) run(ctx context.Context, tenantID, ruleName string, config ProposedRuleConfig, lookback int) (RuleSimulationResult, error) {
	current, err := s.configs.LoadEffectiveRuleConfigs(ctx, tenantID)
	if err != nil {
		return RuleSimulationResult{}, err
	}
	proposed, err := applyProposedConfig(current, ruleName, config)
	if err != nil {
		return RuleSimulationResult{}, err
	}
	events, err := s.loader.Load(ctx, tenantID, time.Duration(lookback)*time.Minute)
	if err != nil {
		return RuleSimulationResult{}, err
	}
	engine := ruleservice.NewRuleEngine(s.fallback)
	engine.SetTenantRuleConfigs(tenantID, proposed)
	result := RuleSimulationResult{TenantID: tenantID, RuleName: ruleName, DryRun: true, LookbackMinutes: lookback}
	now := time.Now().UTC().Add(-time.Duration(lookback) * time.Minute)
	for i, event := range events {
		if !topicApplies(ruleName, event.Topic) {
			continue
		}
		alerts, executions, _ := engine.EvaluateForTenantWithMode(tenantID, event, now.Add(time.Duration(i)*time.Second), ruleservice.EvaluationModeDryRun)
		for _, execution := range executions {
			if execution.RuleName == ruleName && execution.Matched {
				result.MatchedEvents++
			}
		}
		for _, alert := range alerts {
			if alert.AlertType != ruleName {
				continue
			}
			result.WouldTriggerAlerts++
			if len(result.SampleMatches) < MaxSampleMatches {
				result.SampleMatches = append(result.SampleMatches, sampleFromAlert(alert))
			}
		}
	}
	return result, nil
}

func applyProposedConfig(current []domain.RuleConfig, ruleName string, config ProposedRuleConfig) ([]domain.RuleConfig, error) {
	if err := validateProposed(ruleName, config); err != nil {
		return nil, err
	}
	update := config.toUpdate(ruleName)
	if err := validateConfig(ruleName, update); err != nil {
		return nil, err
	}
	proposed := make([]domain.RuleConfig, 0, len(current))
	found := false
	for _, existing := range current {
		copied := existing
		copied.Config = cloneMap(existing.Config)
		if copied.RuleName == ruleName {
			found = true
			overlay(&copied, update)
		}
		proposed = append(proposed, copied)
	}
	if !found {
		return nil, fmt.Errorf("%w: unknown rule name", ErrInvalidRequest)
	}
	return proposed, nil
}

func validateProposed(ruleName string, config ProposedRuleConfig) error {
	if config.Threshold != nil {
		if *config.Threshold < 0 {
			return fmt.Errorf("%w: threshold must be non-negative", ErrInvalidRequest)
		}
		switch ruleName {
		case domain.RuleRapidOrderSubmission, domain.RuleHighCancelRate:
			if math.Abs(*config.Threshold-math.Round(*config.Threshold)) > 0.000001 {
				return fmt.Errorf("%w: threshold must be a whole number for count-based rules", ErrInvalidRequest)
			}
		}
	}
	return nil
}

func overlay(config *domain.RuleConfig, update domain.UpdateRuleConfigRequest) {
	if update.Enabled != nil {
		config.Enabled = *update.Enabled
	}
	if update.Severity != nil {
		config.Severity = strings.ToUpper(strings.TrimSpace(*update.Severity))
	}
	if update.ThresholdNumeric != nil {
		config.ThresholdNumeric = update.ThresholdNumeric
	}
	if update.ThresholdCount != nil {
		config.ThresholdCount = update.ThresholdCount
	}
	if update.WindowSeconds != nil {
		config.WindowSeconds = update.WindowSeconds
	}
	if update.ThresholdPercent != nil {
		config.ThresholdPercent = update.ThresholdPercent
	}
	if update.Config != nil {
		config.Config = cloneMap(update.Config)
	}
	if update.Description != nil {
		config.Description = update.Description
	}
}

func validateRequest(request RuleSimulationRequest, ruleName string) error {
	if strings.TrimSpace(request.TenantID) == "" {
		return fmt.Errorf("%w: tenantId is required", ErrInvalidRequest)
	}
	if ruleName == "" {
		return fmt.Errorf("%w: ruleName is required", ErrInvalidRequest)
	}
	if dryRun := request.DryRun; dryRun != nil && !*dryRun {
		return fmt.Errorf("%w: simulation endpoints only support dryRun=true", ErrInvalidRequest)
	}
	lookback := lookbackMinutes(request.LookbackMinutes)
	if lookback < 1 || lookback > MaxLookbackMinutes {
		return fmt.Errorf("%w: lookbackMinutes must be between 1 and %d", ErrInvalidRequest, MaxLookbackMinutes)
	}
	return nil
}

func validateConfig(ruleName string, request domain.UpdateRuleConfigRequest) error {
	if !domain.IsKnownRule(ruleName) {
		return fmt.Errorf("%w: unknown rule name", ErrInvalidRequest)
	}
	if request.Severity != nil && !validSeverity(*request.Severity) {
		return fmt.Errorf("%w: invalid severity", ErrInvalidRequest)
	}
	if request.ThresholdNumeric != nil && *request.ThresholdNumeric < 0 {
		return fmt.Errorf("%w: thresholdNumeric must be non-negative", ErrInvalidRequest)
	}
	if request.ThresholdCount != nil && *request.ThresholdCount < 0 {
		return fmt.Errorf("%w: thresholdCount must be non-negative", ErrInvalidRequest)
	}
	if request.WindowSeconds != nil && *request.WindowSeconds <= 0 {
		return fmt.Errorf("%w: windowSeconds must be positive", ErrInvalidRequest)
	}
	if request.ThresholdPercent != nil && *request.ThresholdPercent < 0 {
		return fmt.Errorf("%w: thresholdPercent must be non-negative", ErrInvalidRequest)
	}
	return nil
}

func normalizeRuleName(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case strings.ToLower(domain.RuleLargeOrder), "large_order", "large-order":
		return domain.RuleLargeOrder, nil
	case strings.ToLower(domain.RuleRapidOrderSubmission), "rapid_order_submission", "rapid-order-submission", "rapid_order":
		return domain.RuleRapidOrderSubmission, nil
	case strings.ToLower(domain.RuleHighCancelRate), "high_cancel_rate", "high-cancel-rate", "cancel_rate":
		return domain.RuleHighCancelRate, nil
	case strings.ToLower(domain.RuleRiskScoreBreach), "risk_score_breach", "risk-score-breach", "risk_score":
		return domain.RuleRiskScoreBreach, nil
	case strings.ToLower(domain.RuleAbnormalPriceMovement), "abnormal_price_movement", "abnormal-price-movement", "price_spike", "price-spike":
		return domain.RuleAbnormalPriceMovement, nil
	default:
		return "", fmt.Errorf("%w: unknown rule name", ErrInvalidRequest)
	}
}

func topicApplies(ruleName, topic string) bool {
	switch ruleName {
	case domain.RuleLargeOrder:
		return topic == "order.created" || topic == "order.filled"
	case domain.RuleRapidOrderSubmission:
		return topic == "order.created"
	case domain.RuleHighCancelRate:
		return topic == "order.cancelled"
	case domain.RuleRiskScoreBreach:
		return topic == "risk.score.updated"
	case domain.RuleAbnormalPriceMovement:
		return topic == "market.ticks"
	default:
		return false
	}
}

func sampleFromAlert(alert domain.Alert) SampleMatch {
	sample := SampleMatch{Timestamp: alert.CreatedAt, Reason: alert.Description}
	if alert.Symbol != nil {
		sample.Symbol = *alert.Symbol
	}
	sample.ObservedValue = firstNumber(alert.Metadata, "notional", "count", "score", "changePercent", "currentPrice")
	sample.Threshold = firstNumber(alert.Metadata, "threshold", "limit", "thresholdPercent")
	return sample
}

func firstNumber(values map[string]any, keys ...string) float64 {
	for _, key := range keys {
		switch value := values[key].(type) {
		case float64:
			return value
		case float32:
			return float64(value)
		case int:
			return float64(value)
		case int64:
			return float64(value)
		case jsonNumber:
			parsed, _ := value.Float64()
			return parsed
		}
	}
	return 0
}

type jsonNumber interface {
	Float64() (float64, error)
}

func validSeverity(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case domain.SeverityLow, domain.SeverityMedium, domain.SeverityHigh, domain.SeverityCritical:
		return true
	default:
		return false
	}
}

func lookbackMinutes(value *int) int {
	if value == nil {
		return DefaultLookbackMinutes
	}
	return *value
}

func boolPtr(value bool) *bool {
	return &value
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func (s *Service) recordFailure(ctx context.Context, tenantID, ruleName string, lookback int, correlationID string, err error) {
	if ruleName == "" {
		ruleName = "unknown"
	}
	s.metrics.RuleSimulationRequests.WithLabelValues(ruleName, "failed").Inc()
	s.metrics.RuleSimulationFailures.WithLabelValues(ruleName).Inc()
	s.publish(ctx, "surveillance.rule_simulation.failed", tenantID, ruleName, lookback, 0, 0, "failed", err.Error(), correlationID)
}

func (s *Service) publish(ctx context.Context, eventType, tenantID, ruleName string, lookback, matched, alerts int, status, errorMessage, correlationID string) {
	if s.publisher == nil {
		return
	}
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	if err := s.publisher.PublishRuleSimulation(ctx, domain.RuleSimulationEvent{
		EventID:            uuid.NewString(),
		EventType:          eventType,
		EventVersion:       "1.0",
		TenantID:           tenantID,
		CorrelationID:      correlationID,
		RuleName:           ruleName,
		LookbackMinutes:    lookback,
		DryRun:             true,
		MatchedEvents:      matched,
		WouldTriggerAlerts: alerts,
		Status:             status,
		Error:              errorMessage,
		OccurredAt:         time.Now().UTC(),
	}); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
	}
}
