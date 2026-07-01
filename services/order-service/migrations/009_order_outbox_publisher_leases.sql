ALTER TABLE order_outbox
  ADD COLUMN IF NOT EXISTS locked_by TEXT,
  ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS lease_until TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS failed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_order_outbox_claimable
  ON order_outbox(available_at, created_at, id)
  WHERE published_at IS NULL AND status IN ('pending', 'processing');

CREATE INDEX IF NOT EXISTS idx_order_outbox_failed
  ON order_outbox(failed_at DESC)
  WHERE status = 'failed';
