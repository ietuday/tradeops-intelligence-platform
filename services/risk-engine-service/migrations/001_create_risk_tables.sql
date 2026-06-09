CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS risk_scores (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  user_id TEXT NOT NULL,
  score DOUBLE PRECISION NOT NULL,
  level TEXT NOT NULL,
  factors JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS risk_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  user_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  level TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS risk_recommendations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  user_id TEXT NOT NULL,
  recommendation_type TEXT NOT NULL,
  message TEXT NOT NULL,
  severity TEXT NOT NULL,
  context JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS risk_anomalies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  user_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  anomaly_type TEXT NOT NULL,
  severity TEXT NOT NULL,
  value DOUBLE PRECISION NOT NULL,
  z_score DOUBLE PRECISION NOT NULL,
  event_time TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS risk_calculation_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  user_id TEXT NOT NULL,
  calculation_type TEXT NOT NULL,
  duration_ms INTEGER NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE risk_scores ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE risk_events ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE risk_recommendations ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE risk_anomalies ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE risk_calculation_runs ADD COLUMN IF NOT EXISTS tenant_id TEXT;
UPDATE risk_scores SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE risk_events SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE risk_recommendations SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE risk_anomalies SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE risk_calculation_runs SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';

CREATE INDEX IF NOT EXISTS idx_risk_scores_user_created_at ON risk_scores(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_scores_tenant_user_created_at ON risk_scores(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_events_user_created_at ON risk_events(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_events_tenant_user_created_at ON risk_events(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_recommendations_user_created_at ON risk_recommendations(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_recommendations_tenant_user_created_at ON risk_recommendations(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_anomalies_user_created_at ON risk_anomalies(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_anomalies_tenant_user_created_at ON risk_anomalies(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_calculation_runs_user_created_at ON risk_calculation_runs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_calculation_runs_tenant_user_created_at ON risk_calculation_runs(tenant_id, user_id, created_at DESC);
