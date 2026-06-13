CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    event_type TEXT NOT NULL,
    service_name TEXT NOT NULL,
    actor_user_id UUID,
    entity_type TEXT,
    entity_id TEXT,
    action TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'INFO',
    correlation_id TEXT,
    traceparent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created_at ON audit.audit_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_correlation_id ON audit.audit_events(correlation_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit.audit_events(event_type);
