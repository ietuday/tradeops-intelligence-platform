CREATE TABLE IF NOT EXISTS reference_prices (
  tenant_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  price NUMERIC(20, 8) NOT NULL CHECK (price > 0),
  source TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, symbol)
);

CREATE INDEX IF NOT EXISTS idx_reference_prices_updated_at
  ON reference_prices(updated_at DESC);

ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS triggered_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS original_order_type TEXT,
  ADD COLUMN IF NOT EXISTS activated_order_type TEXT,
  ADD COLUMN IF NOT EXISTS stop_trigger_reference_price NUMERIC(20, 8);

CREATE INDEX IF NOT EXISTS idx_orders_stop_trigger_scan
  ON orders (tenant_id, symbol, created_at, id)
  WHERE order_type IN ('STOP', 'STOP_LIMIT')
    AND stop_price IS NOT NULL
    AND triggered_at IS NULL
    AND remaining_quantity > 0
    AND status IN ('accepted', 'partially_filled');
