CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS risk;

CREATE TABLE IF NOT EXISTS risk.risk_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    user_id UUID,
    portfolio_id UUID,
    check_type TEXT NOT NULL,
    score NUMERIC(10, 4),
    status TEXT NOT NULL DEFAULT 'PASSED',
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS risk.risk_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    risk_check_id UUID REFERENCES risk.risk_checks(id),
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'LOW',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    correlation_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_checks_tenant_status ON risk.risk_checks(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_risk_events_tenant_type ON risk.risk_events(tenant_id, event_type);
