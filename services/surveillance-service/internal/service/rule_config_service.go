package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/repository"
)

var ErrInvalidRuleConfig = errors.New("invalid rule config")

type ruleConfigStore interface {
	ListRuleConfigs(context.Context, string) ([]domain.RuleConfig, error)
	GetRuleConfig(context.Context, string, string) (domain.RuleConfig, error)
	UpdateRuleConfig(context.Context, string, string, domain.UpdateRuleConfigRequest, string) (domain.RuleConfig, error)
	SetRuleEnabled(context.Context, string, string, bool, string) (domain.RuleConfig, error)
	EnsureDefaultRuleConfigs(context.Context, string, []domain.RuleConfig) error
}

type RuleConfigService struct {
	store    ruleConfigStore
	engine   *RuleEngine
	producer *kafka.Producer
	metrics  *observability.Metrics
	fallback RuleConfig
}

func NewRuleConfigService(store ruleConfigStore, engine *RuleEngine, producer *kafka.Producer, metrics *observability.Metrics, fallback RuleConfig) *RuleConfigService {
	return &RuleConfigService{store: store, engine: engine, producer: producer, metrics: metrics, fallback: fallback}
}

func (s *RuleConfigService) EnsureDefaults(ctx context.Context, tenantID string) error {
	if err := s.store.EnsureDefaultRuleConfigs(ctx, defaultTenant(tenantID), DefaultDomainRuleConfigs(defaultTenant(tenantID), s.fallback)); err != nil {
		s.metrics.RuleConfigReloads.WithLabelValues("error").Inc()
		return err
	}
	configs, err := s.LoadEffectiveRuleConfigs(ctx, tenantID)
	if err != nil {
		s.metrics.RuleConfigReloads.WithLabelValues("error").Inc()
		return err
	}
	s.engine.SetTenantRuleConfigs(defaultTenant(tenantID), configs)
	s.metrics.RuleConfigCacheEntries.Set(float64(len(configs)))
	s.metrics.RuleConfigReloads.WithLabelValues("success").Inc()
	return nil
}

func (s *RuleConfigService) ListRules(ctx context.Context, user UserContext) ([]domain.RuleConfig, error) {
	if !canView(user.Roles) {
		return nil, ErrForbidden
	}
	return s.LoadEffectiveRuleConfigs(ctx, defaultTenant(user.TenantID))
}

func (s *RuleConfigService) GetRule(ctx context.Context, user UserContext, ruleName string) (domain.RuleConfig, error) {
	if !canView(user.Roles) {
		return domain.RuleConfig{}, ErrForbidden
	}
	if !domain.IsKnownRule(ruleName) {
		return domain.RuleConfig{}, ErrInvalidRuleConfig
	}
	config, err := s.store.GetRuleConfig(ctx, defaultTenant(user.TenantID), ruleName)
	if errors.Is(err, repository.ErrNotFound) {
		return fallbackDomainRuleConfig(defaultTenant(user.TenantID), ruleName, s.fallback), nil
	}
	return config, err
}

func (s *RuleConfigService) UpdateRule(ctx context.Context, user UserContext, ruleName string, request domain.UpdateRuleConfigRequest, correlationID string) (domain.RuleConfig, error) {
	if !canManage(user.Roles) {
		return domain.RuleConfig{}, ErrForbidden
	}
	if err := validateRuleConfigRequest(ruleName, request); err != nil {
		return domain.RuleConfig{}, err
	}
	config, err := s.store.UpdateRuleConfig(ctx, defaultTenant(user.TenantID), ruleName, request, user.UserID)
	if err != nil {
		return domain.RuleConfig{}, err
	}
	s.refreshTenant(ctx, defaultTenant(user.TenantID))
	s.metrics.RuleConfigUpdates.WithLabelValues(ruleName, "updated").Inc()
	s.publishRuleConfigEvent(ctx, "surveillance.rule_config.updated", config, user.UserID, correlationID, changesFromRequest(request))
	return config, nil
}

func (s *RuleConfigService) EnableRule(ctx context.Context, user UserContext, ruleName string, correlationID string) (domain.RuleConfig, error) {
	return s.setEnabled(ctx, user, ruleName, true, "surveillance.rule_config.enabled", "enabled", correlationID)
}

func (s *RuleConfigService) DisableRule(ctx context.Context, user UserContext, ruleName string, correlationID string) (domain.RuleConfig, error) {
	return s.setEnabled(ctx, user, ruleName, false, "surveillance.rule_config.disabled", "disabled", correlationID)
}

func (s *RuleConfigService) LoadEffectiveRuleConfigs(ctx context.Context, tenantID string) ([]domain.RuleConfig, error) {
	tenantID = defaultTenant(tenantID)
	configs, err := s.store.ListRuleConfigs(ctx, tenantID)
	if err != nil {
		return DefaultDomainRuleConfigs(tenantID, s.fallback), nil
	}
	byRule := map[string]domain.RuleConfig{}
	for _, config := range configs {
		byRule[config.RuleName] = config
	}
	if tenantID != "default-tenant" {
		defaults, err := s.store.ListRuleConfigs(ctx, "default-tenant")
		if err == nil {
			for _, config := range defaults {
				if _, ok := byRule[config.RuleName]; !ok {
					config.TenantID = tenantID
					byRule[config.RuleName] = config
				}
			}
		}
	}
	var effective []domain.RuleConfig
	for _, ruleName := range domain.KnownRuleNames() {
		if config, ok := byRule[ruleName]; ok {
			effective = append(effective, config)
			continue
		}
		effective = append(effective, fallbackDomainRuleConfig(tenantID, ruleName, s.fallback))
	}
	return effective, nil
}

func (s *RuleConfigService) setEnabled(ctx context.Context, user UserContext, ruleName string, enabled bool, eventType, action, correlationID string) (domain.RuleConfig, error) {
	if !canManage(user.Roles) {
		return domain.RuleConfig{}, ErrForbidden
	}
	if !domain.IsKnownRule(ruleName) {
		return domain.RuleConfig{}, ErrInvalidRuleConfig
	}
	config, err := s.store.SetRuleEnabled(ctx, defaultTenant(user.TenantID), ruleName, enabled, user.UserID)
	if err != nil {
		return domain.RuleConfig{}, err
	}
	s.refreshTenant(ctx, defaultTenant(user.TenantID))
	s.metrics.RuleConfigUpdates.WithLabelValues(ruleName, action).Inc()
	s.publishRuleConfigEvent(ctx, eventType, config, user.UserID, correlationID, map[string]any{"enabled": enabled})
	return config, nil
}

func (s *RuleConfigService) refreshTenant(ctx context.Context, tenantID string) {
	configs, err := s.LoadEffectiveRuleConfigs(ctx, tenantID)
	if err != nil {
		s.metrics.RuleConfigReloads.WithLabelValues("error").Inc()
		return
	}
	s.engine.SetTenantRuleConfigs(defaultTenant(tenantID), configs)
	s.metrics.RuleConfigCacheEntries.Set(float64(len(configs)))
	s.metrics.RuleConfigReloads.WithLabelValues("success").Inc()
}

func (s *RuleConfigService) publishRuleConfigEvent(ctx context.Context, eventType string, config domain.RuleConfig, updatedBy string, correlationID string, changes map[string]any) {
	if s.producer == nil {
		return
	}
	var updatedByPtr *string
	if updatedBy != "" {
		updatedByPtr = &updatedBy
	}
	if err := s.producer.PublishRuleConfig(ctx, domain.RuleConfigEvent{
		EventID:          uuid.NewString(),
		EventType:        eventType,
		EventVersion:     "1.0",
		TenantID:         defaultTenant(config.TenantID),
		CorrelationID:    correlationID,
		RuleName:         config.RuleName,
		Enabled:          config.Enabled,
		Severity:         config.Severity,
		ThresholdNumeric: config.ThresholdNumeric,
		ThresholdCount:   config.ThresholdCount,
		WindowSeconds:    config.WindowSeconds,
		ThresholdPercent: config.ThresholdPercent,
		UpdatedBy:        updatedByPtr,
		UpdatedAt:        time.Now().UTC(),
		Changes:          changes,
	}); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
	}
}

func validateRuleConfigRequest(ruleName string, request domain.UpdateRuleConfigRequest) error {
	if !domain.IsKnownRule(ruleName) {
		return fmt.Errorf("%w: unknown rule name", ErrInvalidRuleConfig)
	}
	if request.Severity != nil && !validSeverity(*request.Severity) {
		return fmt.Errorf("%w: invalid severity", ErrInvalidRuleConfig)
	}
	if request.ThresholdNumeric != nil && *request.ThresholdNumeric < 0 {
		return fmt.Errorf("%w: thresholdNumeric must be non-negative", ErrInvalidRuleConfig)
	}
	if request.ThresholdCount != nil && *request.ThresholdCount < 0 {
		return fmt.Errorf("%w: thresholdCount must be non-negative", ErrInvalidRuleConfig)
	}
	if request.WindowSeconds != nil && *request.WindowSeconds <= 0 {
		return fmt.Errorf("%w: windowSeconds must be positive", ErrInvalidRuleConfig)
	}
	if request.ThresholdPercent != nil && *request.ThresholdPercent < 0 {
		return fmt.Errorf("%w: thresholdPercent must be non-negative", ErrInvalidRuleConfig)
	}
	return nil
}

func validSeverity(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case domain.SeverityLow, domain.SeverityMedium, domain.SeverityHigh, domain.SeverityCritical:
		return true
	default:
		return false
	}
}

func changesFromRequest(request domain.UpdateRuleConfigRequest) map[string]any {
	changes := map[string]any{}
	if request.Enabled != nil {
		changes["enabled"] = *request.Enabled
	}
	if request.Severity != nil {
		changes["severity"] = strings.ToUpper(strings.TrimSpace(*request.Severity))
	}
	if request.ThresholdNumeric != nil {
		changes["thresholdNumeric"] = *request.ThresholdNumeric
	}
	if request.ThresholdCount != nil {
		changes["thresholdCount"] = *request.ThresholdCount
	}
	if request.WindowSeconds != nil {
		changes["windowSeconds"] = *request.WindowSeconds
	}
	if request.ThresholdPercent != nil {
		changes["thresholdPercent"] = *request.ThresholdPercent
	}
	if request.Config != nil {
		changes["config"] = request.Config
	}
	if request.Description != nil {
		changes["description"] = *request.Description
	}
	return changes
}
