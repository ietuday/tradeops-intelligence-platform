ALTER TABLE orders ADD COLUMN IF NOT EXISTS filled_quantity NUMERIC NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS remaining_quantity NUMERIC;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS average_fill_price NUMERIC;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS time_in_force TEXT NOT NULL DEFAULT 'DAY';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS risk_decision_id UUID;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS last_execution_at TIMESTAMPTZ;

UPDATE orders
SET remaining_quantity = GREATEST(quantity - filled_quantity, 0)
WHERE remaining_quantity IS NULL;

ALTER TABLE orders ALTER COLUMN remaining_quantity SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_quantity_positive_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_quantity_positive_check CHECK (quantity > 0);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_filled_quantity_non_negative_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_filled_quantity_non_negative_check CHECK (filled_quantity >= 0);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_remaining_quantity_non_negative_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_remaining_quantity_non_negative_check CHECK (remaining_quantity >= 0);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_filled_not_above_quantity_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_filled_not_above_quantity_check CHECK (filled_quantity <= quantity);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_version_positive_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_version_positive_check CHECK (version > 0);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_side_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_side_check CHECK (side IN ('BUY', 'SELL'));
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_type_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_type_check CHECK (order_type IN ('MARKET', 'LIMIT', 'STOP', 'STOP_LIMIT', 'STOP_LOSS'));
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_tif_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_tif_check CHECK (time_in_force IN ('DAY', 'GTC', 'GTD', 'IOC', 'FOK'));
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_status_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('created', 'validated', 'risk_pending', 'risk_rejected', 'accepted', 'partially_filled', 'filled', 'rejected', 'cancelled', 'expired'));
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS order_executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL,
  buy_order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  sell_order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  symbol TEXT NOT NULL,
  execution_quantity NUMERIC NOT NULL CHECK (execution_quantity > 0),
  execution_price NUMERIC NOT NULL CHECK (execution_price > 0),
  buyer_user_id TEXT NOT NULL,
  seller_user_id TEXT NOT NULL,
  correlation_id TEXT,
  traceparent TEXT,
  tracestate TEXT,
  executed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_order_executions_tenant_symbol ON order_executions(tenant_id, symbol, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_executions_buy_order ON order_executions(buy_order_id);
CREATE INDEX IF NOT EXISTS idx_order_executions_sell_order ON order_executions(sell_order_id);
CREATE INDEX IF NOT EXISTS idx_order_executions_executed_at ON order_executions(executed_at DESC);

CREATE TABLE IF NOT EXISTS order_outbox (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  event_id UUID NOT NULL UNIQUE,
  event_type TEXT NOT NULL,
  topic TEXT NOT NULL,
  payload JSONB NOT NULL,
  correlation_id TEXT,
  traceparent TEXT,
  tracestate TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_order_outbox_pending ON order_outbox(status, available_at, created_at);
CREATE INDEX IF NOT EXISTS idx_order_outbox_tenant ON order_outbox(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_outbox_aggregate ON order_outbox(aggregate_type, aggregate_id);

CREATE TABLE IF NOT EXISTS order_risk_decisions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL,
  order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  decision TEXT NOT NULL,
  reason_code TEXT,
  reason_message TEXT,
  evaluated_limits JSONB NOT NULL DEFAULT '{}'::jsonb,
  request_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  response_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  correlation_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_order_risk_decisions_order ON order_risk_decisions(order_id);
CREATE INDEX IF NOT EXISTS idx_orders_tenant_symbol_active ON orders(tenant_id, symbol, side, status, created_at)
  WHERE status IN ('accepted', 'partially_filled');
CREATE INDEX IF NOT EXISTS idx_orders_expiry ON orders(status, time_in_force, expires_at)
  WHERE status IN ('accepted', 'partially_filled');
