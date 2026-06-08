CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
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

CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
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
    requested_by UUID NULL,
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(30) NOT NULL DEFAULT 'COMPLETED',
    file_name VARCHAR(255) NULL,
    record_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
