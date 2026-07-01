CREATE TABLE IF NOT EXISTS portfolio_processed_events (
  event_id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  topic TEXT,
  partition INTEGER,
  message_offset BIGINT,
  correlation_id TEXT,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  payload_hash TEXT,
  consumer_version TEXT NOT NULL DEFAULT 'v3.1.2',
  UNIQUE (tenant_id, execution_id)
);

UPDATE portfolios SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
ALTER TABLE portfolios ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE portfolios DROP CONSTRAINT IF EXISTS portfolios_user_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_portfolios_tenant_user_unique ON portfolios(tenant_id, user_id);

CREATE TABLE IF NOT EXISTS portfolio_transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL,
  portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  execution_id TEXT NOT NULL,
  order_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL CHECK (side IN ('BUY', 'SELL')),
  quantity NUMERIC(20, 8) NOT NULL,
  price NUMERIC(20, 8) NOT NULL,
  gross_amount NUMERIC(20, 8) NOT NULL,
  fee_amount NUMERIC(20, 8) NOT NULL DEFAULT 0,
  net_cash_amount NUMERIC(20, 8) NOT NULL,
  currency TEXT NOT NULL DEFAULT 'USD',
  realized_pnl NUMERIC(20, 8) NOT NULL DEFAULT 0,
  correlation_id TEXT,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, portfolio_id, execution_id, side)
);

ALTER TABLE portfolio_holdings ADD COLUMN IF NOT EXISTS total_cost NUMERIC(20, 8) NOT NULL DEFAULT 0;
ALTER TABLE portfolio_holdings ADD COLUMN IF NOT EXISTS realized_pnl NUMERIC(20, 8) NOT NULL DEFAULT 0;
ALTER TABLE portfolio_holdings ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_portfolio_processed_events_execution ON portfolio_processed_events(tenant_id, execution_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_processed_events_processed_at ON portfolio_processed_events(processed_at DESC);
CREATE INDEX IF NOT EXISTS idx_portfolio_transactions_execution ON portfolio_transactions(tenant_id, execution_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_transactions_user_created_at ON portfolio_transactions(tenant_id, user_id, created_at DESC);
