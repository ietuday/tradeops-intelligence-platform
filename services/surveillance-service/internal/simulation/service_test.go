package simulation

import (
	"context"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	ruleservice "github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/service"
)

func TestSimulateLargeOrderUsesProposedConfigWithoutMutatingCurrent(t *testing.T) {
	current := defaultConfigs("demo-tenant")
	provider := &fakeProvider{configs: current}
	threshold := 200000.0
	svc := NewService(provider, nil, observability.NewMetrics(), testRuleDefaults(), NewDemoEventLoader())

	result, err := svc.Simulate(context.Background(), RuleSimulationRequest{
		TenantID: "demo-tenant",
		RuleName: domain.RuleLargeOrder,
		Config:   ProposedRuleConfig{ThresholdNumeric: &threshold},
		DryRun:   boolPtr(true),
	}, "test-correlation")
	if err != nil {
		t.Fatalf("Simulate returned error: %v", err)
	}

	if result.MatchedEvents != 1 || result.WouldTriggerAlerts != 1 {
		t.Fatalf("expected one simulated match/alert, got matches=%d alerts=%d", result.MatchedEvents, result.WouldTriggerAlerts)
	}
	if current[0].ThresholdNumeric == nil || *current[0].ThresholdNumeric != 100000 {
		t.Fatalf("current config was mutated: %+v", current[0].ThresholdNumeric)
	}
}

func TestSimulateRejectsDryRunFalse(t *testing.T) {
	svc := NewService(&fakeProvider{configs: defaultConfigs("demo-tenant")}, nil, observability.NewMetrics(), testRuleDefaults(), NewDemoEventLoader())
	dryRun := false
	_, err := svc.Simulate(context.Background(), RuleSimulationRequest{
		TenantID: "demo-tenant",
		RuleName: domain.RuleLargeOrder,
		DryRun:   &dryRun,
	}, "")
	if err == nil {
		t.Fatal("expected dryRun=false to be rejected")
	}
}

func TestSimulateBulkSupportsRuleAliases(t *testing.T) {
	svc := NewService(&fakeProvider{configs: defaultConfigs("demo-tenant")}, nil, observability.NewMetrics(), testRuleDefaults(), NewDemoEventLoader())
	threshold := 5.0
	result, err := svc.SimulateBulk(context.Background(), RuleSimulationRequest{
		TenantID: "demo-tenant",
		Rules: []RuleSimulation{
			{RuleName: "price_spike", Config: ProposedRuleConfig{Threshold: &threshold}},
		},
		DryRun: boolPtr(true),
	}, "")
	if err != nil {
		t.Fatalf("SimulateBulk returned error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(result.Results))
	}
	if result.Results[0].RuleName != domain.RuleAbnormalPriceMovement {
		t.Fatalf("expected alias to normalize, got %s", result.Results[0].RuleName)
	}
	if result.Results[0].WouldTriggerAlerts == 0 {
		t.Fatal("expected price spike simulation to trigger at least one alert")
	}
}

type fakeProvider struct {
	configs []domain.RuleConfig
}

func (f *fakeProvider) LoadEffectiveRuleConfigs(_ context.Context, _ string) ([]domain.RuleConfig, error) {
	copied := make([]domain.RuleConfig, len(f.configs))
	copy(copied, f.configs)
	return copied, nil
}

func defaultConfigs(tenantID string) []domain.RuleConfig {
	return ruleservice.DefaultDomainRuleConfigs(tenantID, testRuleDefaults())
}

func testRuleDefaults() ruleservice.RuleConfig {
	return ruleservice.RuleConfig{
		LargeOrderThreshold:          100000,
		RapidOrderLimit:              5,
		RapidOrderWindow:             time.Minute,
		HighCancelLimit:              3,
		HighCancelWindow:             5 * time.Minute,
		RiskScoreThreshold:           80,
		AbnormalPriceMovementPercent: 10,
	}
}
