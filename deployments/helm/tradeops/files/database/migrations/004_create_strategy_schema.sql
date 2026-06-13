CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS strategy;

CREATE TABLE IF NOT EXISTS strategy.strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT strategy_name_unique UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS strategy.strategy_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    strategy_id UUID REFERENCES strategy.strategies(id),
    name TEXT NOT NULL,
    rule_type TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT strategy_rule_name_unique UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_strategy_strategies_tenant_status ON strategy.strategies(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_strategy_rules_strategy_id ON strategy.strategy_rules(strategy_id);
