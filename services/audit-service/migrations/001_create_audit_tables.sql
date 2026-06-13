CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(100) NULL,
    event_type VARCHAR(150) NOT NULL,
    service_name VARCHAR(100) NOT NULL,
    actor_user_id UUID NULL,
    actor_role VARCHAR(100) NULL,
    entity_type VARCHAR(100) NULL,
    entity_id VARCHAR(150) NULL,
    action VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(30) NOT NULL DEFAULT 'INFO',
    correlation_id VARCHAR(150) NULL,
    ip_address VARCHAR(100) NULL,
    user_agent TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_event_key VARCHAR(255) NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS id UUID;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(100) NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS event_type VARCHAR(150) NOT NULL DEFAULT 'unknown';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS service_name VARCHAR(100) NOT NULL DEFAULT 'unknown';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS actor_user_id UUID NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS actor_role VARCHAR(100) NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entity_type VARCHAR(100) NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entity_id VARCHAR(150) NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS action VARCHAR(100) NOT NULL DEFAULT 'unknown';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS severity VARCHAR(30) NOT NULL DEFAULT 'INFO';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(150) NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS ip_address VARCHAR(100) NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_agent TEXT NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS source_event_key VARCHAR(255) NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
UPDATE audit_logs SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';

CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created_at ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_actor_user_id ON audit_logs(tenant_id, actor_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_service_name ON audit_logs(service_name);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_user_id ON audit_logs(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_type ON audit_logs(entity_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_id ON audit_logs(entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_severity ON audit_logs(severity);
CREATE INDEX IF NOT EXISTS idx_audit_logs_correlation_id ON audit_logs(correlation_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_source_event_key ON audit_logs(source_event_key) WHERE source_event_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS audit_export_requests (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(100) NULL,
    requested_by UUID NULL,
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(30) NOT NULL DEFAULT 'COMPLETED',
    file_name VARCHAR(255) NULL,
    record_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS id UUID;
ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(100) NULL;
ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS requested_by UUID NULL;
ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS filters JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'COMPLETED';
ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS file_name VARCHAR(255) NULL;
ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS record_count INT NOT NULL DEFAULT 0;
ALTER TABLE audit_export_requests ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
UPDATE audit_export_requests SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
CREATE INDEX IF NOT EXISTS idx_audit_export_requests_tenant_created_at ON audit_export_requests(tenant_id, created_at DESC);
