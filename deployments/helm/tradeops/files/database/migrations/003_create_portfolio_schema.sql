CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS portfolio;

CREATE TABLE IF NOT EXISTS portfolio.portfolios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    user_id UUID NOT NULL,
    name TEXT NOT NULL DEFAULT 'Demo Portfolio',
    cash_balance NUMERIC(20, 2) NOT NULL DEFAULT 100000.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT portfolio_user_unique UNIQUE (tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS portfolio.positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL DEFAULT 'default-tenant',
    portfolio_id UUID NOT NULL REFERENCES portfolio.portfolios(id),
    symbol TEXT NOT NULL,
    quantity NUMERIC(20, 8) NOT NULL DEFAULT 0,
    average_price NUMERIC(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT portfolio_position_unique UNIQUE (portfolio_id, symbol)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_portfolios_tenant_id ON portfolio.portfolios(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_positions_tenant_symbol ON portfolio.positions(tenant_id, symbol);
