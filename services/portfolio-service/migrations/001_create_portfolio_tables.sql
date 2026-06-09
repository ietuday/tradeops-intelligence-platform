CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS portfolios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  user_id TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cash_balances (
  portfolio_id UUID PRIMARY KEY REFERENCES portfolios(id) ON DELETE CASCADE,
  cash_balance NUMERIC NOT NULL,
  realized_pnl NUMERIC NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS portfolio_holdings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity NUMERIC NOT NULL,
  average_buy_price NUMERIC NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (portfolio_id, symbol)
);

CREATE TABLE IF NOT EXISTS portfolio_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  cash_balance NUMERIC NOT NULL,
  holdings_value NUMERIC NOT NULL,
  total_value NUMERIC NOT NULL,
  realized_pnl NUMERIC NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS realized_pnl_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  order_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity NUMERIC NOT NULL,
  fill_price NUMERIC NOT NULL,
  average_buy_price NUMERIC NOT NULL,
  realized_pnl NUMERIC NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  correlation_id TEXT
);

CREATE TABLE IF NOT EXISTS processed_order_events (
  event_id TEXT PRIMARY KEY,
  tenant_id TEXT,
  order_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE portfolios ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE portfolio_holdings ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE portfolio_snapshots ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE realized_pnl_events ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE processed_order_events ADD COLUMN IF NOT EXISTS tenant_id TEXT;

UPDATE portfolios SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE portfolio_holdings SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE portfolio_snapshots SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE realized_pnl_events SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE processed_order_events SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';

CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_user_symbol ON portfolio_holdings(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_portfolios_tenant_user ON portfolios(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_tenant_user_symbol ON portfolio_holdings(tenant_id, user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_portfolio_snapshots_user_created_at ON portfolio_snapshots(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_portfolio_snapshots_tenant_user_created_at ON portfolio_snapshots(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_realized_pnl_events_user_created_at ON realized_pnl_events(user_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_realized_pnl_events_tenant_user_created_at ON realized_pnl_events(tenant_id, user_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_processed_order_events_tenant_id ON processed_order_events(tenant_id);
