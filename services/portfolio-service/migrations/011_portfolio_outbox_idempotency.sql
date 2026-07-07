ALTER TABLE portfolio_processed_events ADD COLUMN IF NOT EXISTS id BIGSERIAL;
ALTER TABLE portfolio_processed_events ADD COLUMN IF NOT EXISTS event_version TEXT NOT NULL DEFAULT 'v1';
ALTER TABLE portfolio_processed_events ADD COLUMN IF NOT EXISTS source_service TEXT NOT NULL DEFAULT 'order-service';
ALTER TABLE portfolio_processed_events ADD COLUMN IF NOT EXISTS aggregate_id TEXT;
ALTER TABLE portfolio_processed_events ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE portfolio_processed_events ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE portfolio_processed_events ALTER COLUMN execution_id DROP NOT NULL;

UPDATE portfolio_processed_events
SET aggregate_id = COALESCE(aggregate_id, execution_id),
    idempotency_key = COALESCE(idempotency_key, 'execution:' || execution_id)
WHERE idempotency_key IS NULL;

ALTER TABLE portfolio_processed_events ALTER COLUMN idempotency_key SET NOT NULL;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'portfolio_processed_events_pkey') THEN
    ALTER TABLE portfolio_processed_events DROP CONSTRAINT portfolio_processed_events_pkey;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'portfolio_processed_events_id_pkey') THEN
    ALTER TABLE portfolio_processed_events ADD CONSTRAINT portfolio_processed_events_id_pkey PRIMARY KEY (id);
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_portfolio_processed_events_tenant_event
  ON portfolio_processed_events(tenant_id, event_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_portfolio_processed_events_tenant_idempotency
  ON portfolio_processed_events(tenant_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_portfolio_processed_events_tenant_type
  ON portfolio_processed_events(tenant_id, event_type);
CREATE INDEX IF NOT EXISTS idx_portfolio_processed_events_tenant_processed
  ON portfolio_processed_events(tenant_id, processed_at);
CREATE INDEX IF NOT EXISTS idx_portfolio_processed_events_tenant_aggregate
  ON portfolio_processed_events(tenant_id, aggregate_id);

CREATE TABLE IF NOT EXISTS portfolio_outbox_events (
  id BIGSERIAL PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE,
  tenant_id TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  event_version TEXT NOT NULL,
  topic TEXT NOT NULL,
  payload JSONB NOT NULL,
  headers JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'publishing', 'published', 'failed')),
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_portfolio_outbox_pending
  ON portfolio_outbox_events(status, next_attempt_at, created_at);
CREATE INDEX IF NOT EXISTS idx_portfolio_outbox_tenant_created
  ON portfolio_outbox_events(tenant_id, created_at);
