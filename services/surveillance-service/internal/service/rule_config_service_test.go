package service

import (
	"context"
	"testing"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/repository"
)

func TestRuleConfigServiceFallbackToDefaultTenant(t *testing.T) {
	store := newFakeRuleConfigStore()
	defaultThreshold := 250000.0
	store.configs["default-tenant"] = map[string]domain.RuleConfig{
		domain.RuleLargeOrder: {
			ID:               "rule-1",
			TenantID:         "default-tenant",
			RuleName:         domain.RuleLargeOrder,
			Enabled:          true,
			Severity:         domain.SeverityCritical,
			ThresholdNumeric: &defaultThreshold,
			Config:           map[string]any{},
		},
	}
	svc := NewRuleConfigService(store, NewRuleEngine(testRuleConfig()), nil, observability.NewMetrics(), testRuleConfig())

	rules, err := svc.LoadEffectiveRuleConfigs(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("load effective configs: %v", err)
	}
	found := findRuleConfig(rules, domain.RuleLargeOrder)
	if found == nil || found.ThresholdNumeric == nil || *found.ThresholdNumeric != defaultThreshold || found.TenantID != "tenant-a" {
		t.Fatalf("expected default-tenant fallback projected to tenant-a, got %+v", found)
	}
}

func TestRuleConfigServiceUpdateAndEnableDisable(t *testing.T) {
	store := newFakeRuleConfigStore()
	svc := NewRuleConfigService(store, NewRuleEngine(testRuleConfig()), nil, observability.NewMetrics(), testRuleConfig())
	if err := svc.EnsureDefaults(context.Background(), "tenant-a"); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	threshold := 150000.0
	updated, err := svc.UpdateRule(context.Background(), adminUser("tenant-a"), domain.RuleLargeOrder, domain.UpdateRuleConfigRequest{ThresholdNumeric: &threshold}, "corr-1")
	if err != nil {
		t.Fatalf("update rule: %v", err)
	}
	if updated.ThresholdNumeric == nil || *updated.ThresholdNumeric != threshold {
		t.Fatalf("expected updated threshold, got %+v", updated)
	}
	disabled, err := svc.DisableRule(context.Background(), adminUser("tenant-a"), domain.RuleLargeOrder, "corr-2")
	if err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected disabled rule, got %+v", disabled)
	}
	enabled, err := svc.EnableRule(context.Background(), adminUser("tenant-a"), domain.RuleLargeOrder, "corr-3")
	if err != nil {
		t.Fatalf("enable rule: %v", err)
	}
	if !enabled.Enabled {
		t.Fatalf("expected enabled rule, got %+v", enabled)
	}
}

func TestRuleConfigServiceRejectsUnauthorizedWrite(t *testing.T) {
	svc := NewRuleConfigService(newFakeRuleConfigStore(), NewRuleEngine(testRuleConfig()), nil, observability.NewMetrics(), testRuleConfig())
	_, err := svc.UpdateRule(context.Background(), UserContext{TenantID: "tenant-a", Roles: []string{"viewer"}}, domain.RuleLargeOrder, domain.UpdateRuleConfigRequest{}, "corr-1")
	if err == nil || err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func adminUser(tenantID string) UserContext {
	return UserContext{UserID: "11111111-1111-1111-1111-111111111111", TenantID: tenantID, Roles: []string{"risk_manager"}}
}

func findRuleConfig(rules []domain.RuleConfig, ruleName string) *domain.RuleConfig {
	for i := range rules {
		if rules[i].RuleName == ruleName {
			return &rules[i]
		}
	}
	return nil
}

type fakeRuleConfigStore struct {
	configs map[string]map[string]domain.RuleConfig
}

func newFakeRuleConfigStore() *fakeRuleConfigStore {
	return &fakeRuleConfigStore{configs: map[string]map[string]domain.RuleConfig{}}
}

func (s *fakeRuleConfigStore) ListRuleConfigs(_ context.Context, tenantID string) ([]domain.RuleConfig, error) {
	var rules []domain.RuleConfig
	for _, config := range s.configs[defaultTenant(tenantID)] {
		rules = append(rules, config)
	}
	return rules, nil
}

func (s *fakeRuleConfigStore) GetRuleConfig(_ context.Context, tenantID, ruleName string) (domain.RuleConfig, error) {
	if config, ok := s.configs[defaultTenant(tenantID)][ruleName]; ok {
		return config, nil
	}
	if defaultTenant(tenantID) != "default-tenant" {
		if config, ok := s.configs["default-tenant"][ruleName]; ok {
			return config, nil
		}
	}
	return domain.RuleConfig{}, repository.ErrNotFound
}

func (s *fakeRuleConfigStore) UpdateRuleConfig(ctx context.Context, tenantID, ruleName string, request domain.UpdateRuleConfigRequest, updatedBy string) (domain.RuleConfig, error) {
	config, err := s.GetRuleConfig(ctx, tenantID, ruleName)
	if err != nil {
		config = fallbackDomainRuleConfig(defaultTenant(tenantID), ruleName, testRuleConfig())
	}
	config.TenantID = defaultTenant(tenantID)
	if request.Enabled != nil {
		config.Enabled = *request.Enabled
	}
	if request.Severity != nil {
		config.Severity = *request.Severity
	}
	if request.ThresholdNumeric != nil {
		config.ThresholdNumeric = request.ThresholdNumeric
	}
	if request.ThresholdCount != nil {
		config.ThresholdCount = request.ThresholdCount
	}
	if request.WindowSeconds != nil {
		config.WindowSeconds = request.WindowSeconds
	}
	if request.ThresholdPercent != nil {
		config.ThresholdPercent = request.ThresholdPercent
	}
	if request.Config != nil {
		config.Config = request.Config
	}
	if request.Description != nil {
		config.Description = request.Description
	}
	if updatedBy != "" {
		config.UpdatedBy = &updatedBy
	}
	s.put(config)
	return config, nil
}

func (s *fakeRuleConfigStore) SetRuleEnabled(ctx context.Context, tenantID, ruleName string, enabled bool, updatedBy string) (domain.RuleConfig, error) {
	return s.UpdateRuleConfig(ctx, tenantID, ruleName, domain.UpdateRuleConfigRequest{Enabled: &enabled}, updatedBy)
}

func (s *fakeRuleConfigStore) EnsureDefaultRuleConfigs(_ context.Context, tenantID string, defaults []domain.RuleConfig) error {
	for _, config := range defaults {
		config.TenantID = defaultTenant(tenantID)
		if _, ok := s.configs[config.TenantID][config.RuleName]; ok {
			continue
		}
		s.put(config)
	}
	return nil
}

func (s *fakeRuleConfigStore) put(config domain.RuleConfig) {
	tenantID := defaultTenant(config.TenantID)
	if s.configs[tenantID] == nil {
		s.configs[tenantID] = map[string]domain.RuleConfig{}
	}
	s.configs[tenantID][config.RuleName] = config
}
