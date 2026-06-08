CREATE TABLE IF NOT EXISTS surveillance_alerts (
    id UUID PRIMARY KEY,
    alert_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    user_id UUID NULL,
    symbol VARCHAR(30) NULL,
    description TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMP WITH TIME ZONE NULL,
    resolved_at TIMESTAMP WITH TIME ZONE NULL,
    dismissed_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE INDEX IF NOT EXISTS idx_surveillance_alerts_status ON surveillance_alerts (status);
CREATE INDEX IF NOT EXISTS idx_surveillance_alerts_severity ON surveillance_alerts (severity);
CREATE INDEX IF NOT EXISTS idx_surveillance_alerts_alert_type ON surveillance_alerts (alert_type);
CREATE INDEX IF NOT EXISTS idx_surveillance_alerts_user_id ON surveillance_alerts (user_id);
CREATE INDEX IF NOT EXISTS idx_surveillance_alerts_symbol ON surveillance_alerts (symbol);
CREATE INDEX IF NOT EXISTS idx_surveillance_alerts_created_at ON surveillance_alerts (created_at DESC);

CREATE TABLE IF NOT EXISTS surveillance_rule_executions (
    id UUID PRIMARY KEY,
    rule_name VARCHAR(100) NOT NULL,
    source_topic VARCHAR(100) NOT NULL,
    entity_id VARCHAR(100) NULL,
    matched BOOLEAN NOT NULL,
    execution_time_ms BIGINT NOT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
