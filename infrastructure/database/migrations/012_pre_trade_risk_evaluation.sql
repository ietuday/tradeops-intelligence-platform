CREATE TABLE IF NOT EXISTS pre_trade_risk_policies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  version TEXT NOT NULL DEFAULT 'pretrade-v1',
  max_order_quantity NUMERIC NOT NULL,
  max_order_notional NUMERIC NOT NULL,
  max_daily_user_notional NUMERIC NOT NULL,
  max_daily_tenant_notional NUMERIC NOT NULL,
  max_open_orders INTEGER,
  allowed_symbols TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  restricted_symbols TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  currency TEXT NOT NULL DEFAULT 'USD',
  risk_timezone TEXT NOT NULL DEFAULT 'UTC',
  effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
  effective_until TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pre_trade_risk_policies_active
  ON pre_trade_risk_policies(tenant_id, enabled, effective_from DESC)
  WHERE enabled = TRUE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_pre_trade_risk_policies_default_version
  ON pre_trade_risk_policies(version)
  WHERE tenant_id IS NULL;

INSERT INTO pre_trade_risk_policies (
  tenant_id, enabled, version, max_order_quantity, max_order_notional,
  max_daily_user_notional, max_daily_tenant_notional, max_open_orders,
  allowed_symbols, restricted_symbols, currency, risk_timezone
)
SELECT NULL, TRUE, 'pretrade-v1', 1000, 250000, 500000, 5000000, 100,
       ARRAY[]::TEXT[], ARRAY[]::TEXT[], 'USD', 'UTC'
WHERE NOT EXISTS (
  SELECT 1 FROM pre_trade_risk_policies WHERE tenant_id IS NULL AND version = 'pretrade-v1'
);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS risk_status TEXT NOT NULL DEFAULT 'NOT_EVALUATED';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS risk_reason_code TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS risk_reason_message TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS risk_evaluated_at TIMESTAMPTZ;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_status_check') THEN
    ALTER TABLE orders DROP CONSTRAINT orders_status_check;
  END IF;
  ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('created', 'validated', 'risk_pending', 'risk_rejected', 'risk_error', 'accepted', 'partially_filled', 'filled', 'rejected', 'cancelled', 'expired'));

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_risk_status_check') THEN
    ALTER TABLE orders ADD CONSTRAINT orders_risk_status_check CHECK (risk_status IN ('NOT_EVALUATED', 'PENDING', 'APPROVED', 'REJECTED', 'ERROR'));
  END IF;
END $$;

ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS user_id TEXT;
ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS policy_id UUID;
ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS policy_version TEXT;
ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS approved BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS estimated_price NUMERIC;
ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS estimated_notional NUMERIC;
ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS traceparent TEXT;
ALTER TABLE order_risk_decisions ADD COLUMN IF NOT EXISTS evaluated_at TIMESTAMPTZ;

UPDATE order_risk_decisions SET user_id = o.user_id
FROM orders o
WHERE order_risk_decisions.order_id = o.id AND order_risk_decisions.user_id IS NULL;
UPDATE order_risk_decisions SET policy_version = 'pretrade-v1' WHERE policy_version IS NULL;
UPDATE order_risk_decisions SET evaluated_at = created_at WHERE evaluated_at IS NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'order_risk_decisions_decision_check') THEN
    ALTER TABLE order_risk_decisions ADD CONSTRAINT order_risk_decisions_decision_check CHECK (decision IN ('APPROVED', 'REJECTED', 'UNAVAILABLE', 'INVALID_RESPONSE'));
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_order_risk_decisions_one_final_per_order
  ON order_risk_decisions(order_id)
  WHERE decision IN ('APPROVED', 'REJECTED', 'UNAVAILABLE', 'INVALID_RESPONSE');
CREATE INDEX IF NOT EXISTS idx_order_risk_decisions_tenant_order ON order_risk_decisions(tenant_id, order_id);
CREATE INDEX IF NOT EXISTS idx_order_risk_decisions_user_day ON order_risk_decisions(tenant_id, user_id, evaluated_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_risk_pending ON orders(tenant_id, status, created_at)
  WHERE status = 'risk_pending';
