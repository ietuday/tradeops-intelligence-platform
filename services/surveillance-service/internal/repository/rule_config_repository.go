package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RuleConfigRepository struct {
	db *pgxpool.Pool
}

func NewRuleConfigRepository(db *pgxpool.Pool) *RuleConfigRepository {
	return &RuleConfigRepository{db: db}
}

func (r *RuleConfigRepository) ListRuleConfigs(ctx context.Context, tenantID string) ([]domain.RuleConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, tenant_id, rule_name, enabled, severity, threshold_numeric, threshold_count, window_seconds,
		       threshold_percent, config, description, updated_by::text, created_at, updated_at
		FROM surveillance_rule_configs
		WHERE tenant_id = $1
		ORDER BY rule_name
	`, defaultTenant(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuleConfigs(rows)
}

func (r *RuleConfigRepository) GetRuleConfig(ctx context.Context, tenantID, ruleName string) (domain.RuleConfig, error) {
	config, err := r.getRuleConfigExact(ctx, defaultTenant(tenantID), ruleName)
	if err == nil {
		return config, nil
	}
	if !errors.Is(err, ErrNotFound) || defaultTenant(tenantID) == "default-tenant" {
		return domain.RuleConfig{}, err
	}
	return r.getRuleConfigExact(ctx, "default-tenant", ruleName)
}

func (r *RuleConfigRepository) UpsertRuleConfig(ctx context.Context, config domain.RuleConfig) (domain.RuleConfig, error) {
	if config.ID == "" {
		config.ID = uuid.NewString()
	}
	if config.Config == nil {
		config.Config = map[string]any{}
	}
	metadata, err := json.Marshal(config.Config)
	if err != nil {
		return domain.RuleConfig{}, err
	}
	return r.scanRuleConfig(ctx, `
		INSERT INTO surveillance_rule_configs (
			id, tenant_id, rule_name, enabled, severity, threshold_numeric, threshold_count,
			window_seconds, threshold_percent, config, description, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12::uuid)
		ON CONFLICT (tenant_id, rule_name) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			severity = EXCLUDED.severity,
			threshold_numeric = EXCLUDED.threshold_numeric,
			threshold_count = EXCLUDED.threshold_count,
			window_seconds = EXCLUDED.window_seconds,
			threshold_percent = EXCLUDED.threshold_percent,
			config = EXCLUDED.config,
			description = EXCLUDED.description,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
		RETURNING id::text, tenant_id, rule_name, enabled, severity, threshold_numeric, threshold_count,
		          window_seconds, threshold_percent, config, description, updated_by::text, created_at, updated_at
	`, config.ID, defaultTenant(config.TenantID), config.RuleName, config.Enabled, config.Severity, config.ThresholdNumeric, config.ThresholdCount, config.WindowSeconds, config.ThresholdPercent, string(metadata), config.Description, config.UpdatedBy)
}

func (r *RuleConfigRepository) UpdateRuleConfig(ctx context.Context, tenantID, ruleName string, request domain.UpdateRuleConfigRequest, updatedBy string) (domain.RuleConfig, error) {
	current, err := r.GetRuleConfig(ctx, tenantID, ruleName)
	if err != nil {
		return domain.RuleConfig{}, err
	}
	current.TenantID = defaultTenant(tenantID)
	if request.Enabled != nil {
		current.Enabled = *request.Enabled
	}
	if request.Severity != nil {
		current.Severity = strings.ToUpper(strings.TrimSpace(*request.Severity))
	}
	if request.ThresholdNumeric != nil {
		current.ThresholdNumeric = request.ThresholdNumeric
	}
	if request.ThresholdCount != nil {
		current.ThresholdCount = request.ThresholdCount
	}
	if request.WindowSeconds != nil {
		current.WindowSeconds = request.WindowSeconds
	}
	if request.ThresholdPercent != nil {
		current.ThresholdPercent = request.ThresholdPercent
	}
	if request.Config != nil {
		current.Config = request.Config
	}
	if request.Description != nil {
		current.Description = request.Description
	}
	if updatedBy != "" {
		current.UpdatedBy = &updatedBy
	}
	return r.UpsertRuleConfig(ctx, current)
}

func (r *RuleConfigRepository) SetRuleEnabled(ctx context.Context, tenantID, ruleName string, enabled bool, updatedBy string) (domain.RuleConfig, error) {
	return r.UpdateRuleConfig(ctx, tenantID, ruleName, domain.UpdateRuleConfigRequest{Enabled: &enabled}, updatedBy)
}

func (r *RuleConfigRepository) EnsureDefaultRuleConfigs(ctx context.Context, tenantID string, defaults []domain.RuleConfig) error {
	for _, config := range defaults {
		config.TenantID = defaultTenant(tenantID)
		if config.ID == "" {
			config.ID = uuid.NewString()
		}
		if config.Config == nil {
			config.Config = map[string]any{}
		}
		metadata, err := json.Marshal(config.Config)
		if err != nil {
			return err
		}
		_, err = r.db.Exec(ctx, `
			INSERT INTO surveillance_rule_configs (
				id, tenant_id, rule_name, enabled, severity, threshold_numeric, threshold_count,
				window_seconds, threshold_percent, config, description
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)
			ON CONFLICT (tenant_id, rule_name) DO NOTHING
		`, config.ID, config.TenantID, config.RuleName, config.Enabled, config.Severity, config.ThresholdNumeric, config.ThresholdCount, config.WindowSeconds, config.ThresholdPercent, string(metadata), config.Description)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RuleConfigRepository) getRuleConfigExact(ctx context.Context, tenantID, ruleName string) (domain.RuleConfig, error) {
	return r.scanRuleConfig(ctx, `
		SELECT id::text, tenant_id, rule_name, enabled, severity, threshold_numeric, threshold_count, window_seconds,
		       threshold_percent, config, description, updated_by::text, created_at, updated_at
		FROM surveillance_rule_configs
		WHERE tenant_id = $1 AND rule_name = $2
	`, defaultTenant(tenantID), ruleName)
}

func (r *RuleConfigRepository) scanRuleConfig(ctx context.Context, query string, args ...any) (domain.RuleConfig, error) {
	var config domain.RuleConfig
	var rawConfig []byte
	var thresholdNumeric sql.NullFloat64
	var thresholdCount sql.NullInt64
	var windowSeconds sql.NullInt64
	var thresholdPercent sql.NullFloat64
	var description sql.NullString
	var updatedBy sql.NullString
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&config.ID,
		&config.TenantID,
		&config.RuleName,
		&config.Enabled,
		&config.Severity,
		&thresholdNumeric,
		&thresholdCount,
		&windowSeconds,
		&thresholdPercent,
		&rawConfig,
		&description,
		&updatedBy,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RuleConfig{}, ErrNotFound
	}
	if err != nil {
		return domain.RuleConfig{}, err
	}
	return hydrateRuleConfig(config, thresholdNumeric, thresholdCount, windowSeconds, thresholdPercent, rawConfig, description, updatedBy), nil
}

func scanRuleConfigs(rows pgx.Rows) ([]domain.RuleConfig, error) {
	var configs []domain.RuleConfig
	for rows.Next() {
		var config domain.RuleConfig
		var rawConfig []byte
		var thresholdNumeric sql.NullFloat64
		var thresholdCount sql.NullInt64
		var windowSeconds sql.NullInt64
		var thresholdPercent sql.NullFloat64
		var description sql.NullString
		var updatedBy sql.NullString
		if err := rows.Scan(&config.ID, &config.TenantID, &config.RuleName, &config.Enabled, &config.Severity, &thresholdNumeric, &thresholdCount, &windowSeconds, &thresholdPercent, &rawConfig, &description, &updatedBy, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, hydrateRuleConfig(config, thresholdNumeric, thresholdCount, windowSeconds, thresholdPercent, rawConfig, description, updatedBy))
	}
	return configs, rows.Err()
}

func hydrateRuleConfig(config domain.RuleConfig, thresholdNumeric sql.NullFloat64, thresholdCount, windowSeconds sql.NullInt64, thresholdPercent sql.NullFloat64, rawConfig []byte, description, updatedBy sql.NullString) domain.RuleConfig {
	if thresholdNumeric.Valid {
		config.ThresholdNumeric = &thresholdNumeric.Float64
	}
	if thresholdCount.Valid {
		value := int(thresholdCount.Int64)
		config.ThresholdCount = &value
	}
	if windowSeconds.Valid {
		value := int(windowSeconds.Int64)
		config.WindowSeconds = &value
	}
	if thresholdPercent.Valid {
		config.ThresholdPercent = &thresholdPercent.Float64
	}
	if description.Valid {
		config.Description = &description.String
	}
	if updatedBy.Valid {
		config.UpdatedBy = &updatedBy.String
	}
	if err := json.Unmarshal(rawConfig, &config.Config); err != nil || config.Config == nil {
		config.Config = map[string]any{}
	}
	return config
}
