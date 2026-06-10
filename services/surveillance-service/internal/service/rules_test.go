package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/repository"
)

func TestRuleEngineDetectsLargeOrder(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	alerts, executions := engine.Evaluate(sourceEvent("order.filled", map[string]any{
		"orderId":   "order-1",
		"userId":    "11111111-1111-1111-1111-111111111111",
		"symbol":    "AAPL",
		"quantity":  1000,
		"fillPrice": 150,
	}), time.Now().UTC())

	if len(alerts) != 1 {
		t.Fatalf("expected one alert, got %d", len(alerts))
	}
	if alerts[0].AlertType != "LargeOrderRule" || alerts[0].Severity != domain.SeverityHigh {
		t.Fatalf("unexpected alert: %+v", alerts[0])
	}
	if len(executions) != 1 || !executions[0].Matched {
		t.Fatalf("expected matched execution, got %+v", executions)
	}
}

func TestRuleEngineDetectsRapidOrderSubmission(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		alerts, _ := engine.Evaluate(sourceEvent("order.created", map[string]any{
			"orderId": "order-rapid",
			"userId":  "11111111-1111-1111-1111-111111111111",
			"symbol":  "AAPL",
		}), now.Add(time.Duration(i)*time.Second))
		if len(alerts) != 0 {
			t.Fatalf("did not expect alert before limit, got %+v", alerts)
		}
	}

	alerts, _ := engine.Evaluate(sourceEvent("order.created", map[string]any{
		"orderId": "order-rapid-6",
		"userId":  "11111111-1111-1111-1111-111111111111",
		"symbol":  "AAPL",
	}), now.Add(6*time.Second))

	if len(alerts) != 1 || alerts[0].AlertType != "RapidOrderSubmissionRule" {
		t.Fatalf("expected rapid order alert, got %+v", alerts)
	}
}

func TestRuleEngineDetectsHighCancelRate(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		engine.Evaluate(sourceEvent("order.cancelled", map[string]any{
			"orderId": "order-cancel",
			"userId":  "11111111-1111-1111-1111-111111111111",
		}), now.Add(time.Duration(i)*time.Second))
	}

	alerts, _ := engine.Evaluate(sourceEvent("order.cancelled", map[string]any{
		"orderId": "order-cancel-4",
		"userId":  "11111111-1111-1111-1111-111111111111",
	}), now.Add(4*time.Second))

	if len(alerts) != 1 || alerts[0].AlertType != "HighCancelRateRule" {
		t.Fatalf("expected high cancel alert, got %+v", alerts)
	}
}

func TestRuleEngineDetectsRiskScoreBreach(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	alerts, _ := engine.Evaluate(sourceEvent("risk.score.updated", map[string]any{
		"userId": "11111111-1111-1111-1111-111111111111",
		"score":  85,
	}), time.Now().UTC())

	if len(alerts) != 1 || alerts[0].AlertType != "RiskScoreBreachRule" || alerts[0].Severity != domain.SeverityCritical {
		t.Fatalf("expected risk score alert, got %+v", alerts)
	}
}

func TestRuleEngineDetectsAbnormalPriceMovement(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	now := time.Now().UTC()
	engine.Evaluate(sourceEvent("market.ticks", map[string]any{
		"symbol": "AAPL",
		"price":  100,
	}), now)

	alerts, _ := engine.Evaluate(sourceEvent("market.ticks", map[string]any{
		"symbol": "AAPL",
		"price":  112,
	}), now.Add(time.Second))

	if len(alerts) != 1 || alerts[0].AlertType != "AbnormalPriceMovementRule" {
		t.Fatalf("expected abnormal price movement alert, got %+v", alerts)
	}
}

func TestRuleEngineSkipsDisabledRule(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	engine.SetTenantRuleConfigs("tenant-a", []domain.RuleConfig{{
		RuleName: domain.RuleLargeOrder,
		Enabled:  false,
		Severity: domain.SeverityHigh,
	}})

	alerts, executions, skipped := engine.EvaluateForTenant("tenant-a", sourceEvent("order.filled", map[string]any{
		"orderId":   "order-disabled",
		"userId":    "11111111-1111-1111-1111-111111111111",
		"symbol":    "AAPL",
		"quantity":  1000,
		"fillPrice": 150,
	}), time.Now().UTC())

	if len(alerts) != 0 || len(executions) != 0 {
		t.Fatalf("disabled rule should not alert or execute, alerts=%+v executions=%+v", alerts, executions)
	}
	if len(skipped) != 1 || skipped[0] != domain.RuleLargeOrder {
		t.Fatalf("expected disabled skip for LargeOrderRule, got %+v", skipped)
	}
}

func TestRuleEngineUsesTenantRuleThresholdAndSeverity(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	threshold := 200000.0
	engine.SetTenantRuleConfigs("tenant-a", []domain.RuleConfig{{
		RuleName:         domain.RuleLargeOrder,
		Enabled:          true,
		Severity:         domain.SeverityCritical,
		ThresholdNumeric: &threshold,
	}})

	alerts, _, _ := engine.EvaluateForTenant("tenant-a", sourceEvent("order.filled", map[string]any{
		"orderId":   "order-under-config",
		"userId":    "11111111-1111-1111-1111-111111111111",
		"symbol":    "AAPL",
		"quantity":  1000,
		"fillPrice": 150,
	}), time.Now().UTC())
	if len(alerts) != 0 {
		t.Fatalf("did not expect alert below tenant threshold, got %+v", alerts)
	}

	alerts, _, _ = engine.EvaluateForTenant("tenant-a", sourceEvent("order.filled", map[string]any{
		"orderId":   "order-over-config",
		"userId":    "11111111-1111-1111-1111-111111111111",
		"symbol":    "AAPL",
		"quantity":  2000,
		"fillPrice": 150,
	}), time.Now().UTC())
	if len(alerts) != 1 || alerts[0].Severity != domain.SeverityCritical {
		t.Fatalf("expected critical alert over tenant threshold, got %+v", alerts)
	}
}

func TestRuleEngineUsesConfiguredRollingWindows(t *testing.T) {
	engine := NewRuleEngine(testRuleConfig())
	limit := 1
	window := 10
	engine.SetTenantRuleConfigs("tenant-a", []domain.RuleConfig{{
		RuleName:       domain.RuleRapidOrderSubmission,
		Enabled:        true,
		Severity:       domain.SeverityHigh,
		ThresholdCount: &limit,
		WindowSeconds:  &window,
	}})
	now := time.Now().UTC()
	engine.EvaluateForTenant("tenant-a", sourceEvent("order.created", map[string]any{
		"orderId": "order-1",
		"userId":  "11111111-1111-1111-1111-111111111111",
	}), now)
	alerts, _, _ := engine.EvaluateForTenant("tenant-a", sourceEvent("order.created", map[string]any{
		"orderId": "order-2",
		"userId":  "11111111-1111-1111-1111-111111111111",
	}), now.Add(time.Second))
	if len(alerts) != 1 || alerts[0].AlertType != domain.RuleRapidOrderSubmission {
		t.Fatalf("expected rapid order alert from configured limit/window, got %+v", alerts)
	}
}

func TestProcessEventSkipsDuplicateAlert(t *testing.T) {
	store := &fakeAlertStore{duplicate: true}
	svc := NewSurveillanceService(store, nil, observability.NewMetrics(), NewRuleEngine(testRuleConfig()))

	err := svc.ProcessEvent(context.Background(), sourceEvent("order.filled", map[string]any{
		"orderId":   "order-1",
		"userId":    "11111111-1111-1111-1111-111111111111",
		"symbol":    "AAPL",
		"quantity":  1000,
		"fillPrice": 150,
	}))
	if err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if store.createdCount != 0 {
		t.Fatalf("duplicate alert should not be created, got %d creates", store.createdCount)
	}
	if store.executionCount != 1 {
		t.Fatalf("expected rule execution to be recorded, got %d", store.executionCount)
	}
}

func TestRuleConfigValidationRejectsInvalidValues(t *testing.T) {
	severity := "NOPE"
	if err := validateRuleConfigRequest(domain.RuleLargeOrder, domain.UpdateRuleConfigRequest{Severity: &severity}); err == nil {
		t.Fatal("expected invalid severity error")
	}
	negative := -1.0
	if err := validateRuleConfigRequest(domain.RuleLargeOrder, domain.UpdateRuleConfigRequest{ThresholdNumeric: &negative}); err == nil {
		t.Fatal("expected negative threshold error")
	}
	window := 0
	if err := validateRuleConfigRequest(domain.RuleRapidOrderSubmission, domain.UpdateRuleConfigRequest{WindowSeconds: &window}); err == nil {
		t.Fatal("expected invalid window error")
	}
	if err := validateRuleConfigRequest("UnknownRule", domain.UpdateRuleConfigRequest{}); err == nil {
		t.Fatal("expected unknown rule error")
	}
}

func TestProcessEventAppliesTenantID(t *testing.T) {
	store := &fakeAlertStore{}
	svc := NewSurveillanceService(store, nil, observability.NewMetrics(), NewRuleEngine(testRuleConfig()))

	err := svc.ProcessEvent(context.Background(), sourceEvent("order.filled", map[string]any{
		"tenantId":  "tenant-a",
		"orderId":   "order-1",
		"userId":    "11111111-1111-1111-1111-111111111111",
		"symbol":    "AAPL",
		"quantity":  1000,
		"fillPrice": 150,
	}))
	if err != nil {
		t.Fatalf("process event failed: %v", err)
	}
	if store.alert.TenantID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", store.alert.TenantID)
	}
}

func testRuleConfig() RuleConfig {
	return RuleConfig{
		LargeOrderThreshold:          100000,
		RapidOrderLimit:              5,
		RapidOrderWindow:             60 * time.Second,
		HighCancelLimit:              3,
		HighCancelWindow:             5 * time.Minute,
		RiskScoreThreshold:           80,
		AbnormalPriceMovementPercent: 10,
	}
}

func sourceEvent(topic string, payload map[string]any) domain.SourceEvent {
	data, _ := json.Marshal(payload)
	return domain.SourceEvent{Topic: topic, Value: data}
}

type fakeAlertStore struct {
	duplicate      bool
	createdCount   int
	executionCount int
	alert          domain.Alert
}

func (s *fakeAlertStore) SaveExecution(context.Context, domain.RuleExecution) error {
	s.executionCount++
	return nil
}

func (s *fakeAlertStore) DuplicateAlertExists(context.Context, domain.Alert) (bool, error) {
	return s.duplicate, nil
}

func (s *fakeAlertStore) CreateAlert(_ context.Context, alert domain.Alert) (domain.Alert, error) {
	s.createdCount++
	s.alert = alert
	return alert, nil
}

func (s *fakeAlertStore) ListAlerts(context.Context, repository.AlertFilters) ([]domain.Alert, error) {
	return []domain.Alert{s.alert}, nil
}

func (s *fakeAlertStore) GetAlert(context.Context, string, string) (domain.Alert, error) {
	return s.alert, nil
}

func (s *fakeAlertStore) UpdateStatus(_ context.Context, _, _ string, status string) (domain.Alert, error) {
	s.alert.Status = status
	return s.alert, nil
}

func (s *fakeAlertStore) Summary(context.Context, string) (repository.Summary, error) {
	return repository.Summary{}, nil
}
