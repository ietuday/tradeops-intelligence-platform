INSERT INTO identity.users (id, tenant_id, email, display_name, password_hash, status)
VALUES
    ('00000000-0000-0000-0000-000000000101', 'default-tenant', 'demo.trader@tradeops.local', 'Demo Trader', 'demo-password-placeholder-not-for-production', 'ACTIVE'),
    ('00000000-0000-0000-0000-000000000102', 'default-tenant', 'demo.compliance@tradeops.local', 'Demo Compliance Officer', 'demo-password-placeholder-not-for-production', 'ACTIVE')
ON CONFLICT (tenant_id, email) DO UPDATE
SET display_name = EXCLUDED.display_name,
    status = EXCLUDED.status,
    updated_at = NOW();

INSERT INTO identity.user_roles (user_id, role_id)
SELECT u.id, r.id
FROM identity.users u
JOIN identity.roles r ON r.name = 'trader'
WHERE u.email = 'demo.trader@tradeops.local'
  AND u.tenant_id = 'default-tenant'
ON CONFLICT (user_id, role_id) DO NOTHING;

INSERT INTO identity.user_roles (user_id, role_id)
SELECT u.id, r.id
FROM identity.users u
JOIN identity.roles r ON r.name IN ('compliance_officer', 'trading_admin')
WHERE u.email = 'demo.compliance@tradeops.local'
  AND u.tenant_id = 'default-tenant'
ON CONFLICT (user_id, role_id) DO NOTHING;
