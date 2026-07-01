ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS expired_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS expiry_reason TEXT,
  ADD COLUMN IF NOT EXISTS expiry_worker_id TEXT;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_expiry_reason_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_expiry_reason_check
      CHECK (expiry_reason IS NULL OR expiry_reason IN ('DAY_SESSION_CLOSED', 'GTD_EXPIRY_REACHED', 'MANUAL_RECONCILIATION', 'LEGACY_EXPIRY_BACKFILL'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_orders_expiry_worker_scan
  ON orders (expires_at, status, id)
  WHERE expires_at IS NOT NULL
    AND remaining_quantity > 0
    AND status IN ('accepted', 'partially_filled')
    AND time_in_force IN ('DAY', 'GTD');
