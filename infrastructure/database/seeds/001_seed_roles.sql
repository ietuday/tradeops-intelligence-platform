INSERT INTO identity.roles (name, description)
VALUES
    ('trader', 'Demo trader role for local TradeOps workflows'),
    ('risk_analyst', 'Demo risk analyst role for local TradeOps workflows'),
    ('compliance_officer', 'Demo compliance role for surveillance and audit workflows'),
    ('trading_admin', 'Demo administrator role for tenant support workflows')
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    updated_at = NOW();
