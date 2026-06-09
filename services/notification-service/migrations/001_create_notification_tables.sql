CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(100) NULL,
    user_id UUID NULL,
    channel VARCHAR(30) NOT NULL,
    priority VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    source VARCHAR(100) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    read_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_tenant_user_created_at ON notifications (tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_tenant_status ON notifications (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications (status);
CREATE INDEX IF NOT EXISTS idx_notifications_channel ON notifications (channel);
CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications (priority);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications (created_at DESC);

CREATE TABLE IF NOT EXISTS notification_delivery_attempts (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(100) NULL,
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL,
    attempt_number INT NOT NULL,
    error_message TEXT NULL,
    attempted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_delivery_attempts_notification_id ON notification_delivery_attempts (notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_attempts_tenant_id ON notification_delivery_attempts (tenant_id);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_attempts_status ON notification_delivery_attempts (status);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_attempts_attempted_at ON notification_delivery_attempts (attempted_at DESC);

CREATE TABLE IF NOT EXISTS notification_preferences (
    tenant_id VARCHAR(100) NULL,
    user_id UUID PRIMARY KEY,
    in_app_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    webhook_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    email_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    webhook_url TEXT NULL,
    email_address TEXT NULL,
    min_priority VARCHAR(30) NOT NULL DEFAULT 'LOW',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(100) NULL;
ALTER TABLE notification_delivery_attempts ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(100) NULL;
ALTER TABLE notification_preferences ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(100) NULL;
UPDATE notifications SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE notification_delivery_attempts SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE notification_preferences SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
CREATE INDEX IF NOT EXISTS idx_notification_preferences_tenant_user ON notification_preferences (tenant_id, user_id);
