CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS portfolios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
  order_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_user_symbol ON portfolio_holdings(user_id, symbol);
CREATE INDEX IF NOT EXISTS idx_portfolio_snapshots_user_created_at ON portfolio_snapshots(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_realized_pnl_events_user_created_at ON realized_pnl_events(user_id, occurred_at DESC);
