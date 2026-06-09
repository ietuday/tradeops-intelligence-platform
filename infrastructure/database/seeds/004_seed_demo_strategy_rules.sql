INSERT INTO strategy.strategies (id, tenant_id, name, description, status, created_by)
VALUES (
    '00000000-0000-0000-0000-000000000301',
    'default-tenant',
    'Demo Momentum Strategy',
    'Demo-only strategy for local migration and seed validation.',
    'ACTIVE',
    '00000000-0000-0000-0000-000000000101'
)
ON CONFLICT (tenant_id, name) DO UPDATE
SET description = EXCLUDED.description,
    status = EXCLUDED.status,
    updated_at = NOW();

INSERT INTO strategy.strategy_rules (tenant_id, strategy_id, name, rule_type, config, enabled)
VALUES
    ('default-tenant', '00000000-0000-0000-0000-000000000301', 'Demo Max Order Notional', 'RISK_LIMIT', '{"maxNotional": 100000}'::jsonb, TRUE),
    ('default-tenant', '00000000-0000-0000-0000-000000000301', 'Demo Momentum Signal', 'SIGNAL', '{"lookbackMinutes": 15, "thresholdPercent": 1.5}'::jsonb, TRUE)
ON CONFLICT (tenant_id, name) DO UPDATE
SET config = EXCLUDED.config,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();
