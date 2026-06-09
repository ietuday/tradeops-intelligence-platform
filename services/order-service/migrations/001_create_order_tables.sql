CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  user_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  order_type TEXT NOT NULL,
  quantity NUMERIC NOT NULL,
  limit_price NUMERIC,
  stop_price NUMERIC,
  status TEXT NOT NULL,
  fill_price NUMERIC,
  reject_reason TEXT,
  correlation_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  cancelled_at TIMESTAMPTZ,
  filled_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS order_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL UNIQUE,
  event_type TEXT NOT NULL,
  tenant_id TEXT,
  order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  payload JSONB NOT NULL,
  correlation_id TEXT,
  occurred_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  tenant_id TEXT,
  user_id TEXT NOT NULL,
  key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, user_id, key)
);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE order_events ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE idempotency_keys ADD COLUMN IF NOT EXISTS tenant_id TEXT;

UPDATE orders SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE order_events SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';
UPDATE idempotency_keys SET tenant_id = 'default-tenant' WHERE tenant_id IS NULL OR tenant_id = '';

CREATE INDEX IF NOT EXISTS idx_orders_user_created_at ON orders(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_tenant_created_at ON orders(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_tenant_user_created_at ON orders(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_tenant_status ON orders(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_order_events_order_id ON order_events(order_id);
CREATE INDEX IF NOT EXISTS idx_order_events_event_type ON order_events(event_type);
CREATE INDEX IF NOT EXISTS idx_order_events_tenant_id ON order_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_tenant_user_key ON idempotency_keys(tenant_id, user_id, key);
