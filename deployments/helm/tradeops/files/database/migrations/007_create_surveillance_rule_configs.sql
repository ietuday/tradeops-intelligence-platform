CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS surveillance_rule_configs (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(100) NOT NULL DEFAULT 'default-tenant',
    rule_name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    severity VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    threshold_numeric DOUBLE PRECISION NULL,
    threshold_count INT NULL,
    window_seconds INT NULL,
    threshold_percent DOUBLE PRECISION NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    description TEXT NULL,
    updated_by UUID NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_surveillance_rule_configs_tenant_rule UNIQUE (tenant_id, rule_name)
);

CREATE INDEX IF NOT EXISTS idx_surveillance_rule_configs_tenant_id ON surveillance_rule_configs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_surveillance_rule_configs_rule_name ON surveillance_rule_configs (rule_name);
CREATE INDEX IF NOT EXISTS idx_surveillance_rule_configs_enabled ON surveillance_rule_configs (enabled);
