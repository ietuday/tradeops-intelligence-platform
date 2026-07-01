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
