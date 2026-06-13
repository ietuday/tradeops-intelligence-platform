INSERT INTO portfolio.portfolios (id, tenant_id, user_id, name, cash_balance)
VALUES (
    '00000000-0000-0000-0000-000000000201',
    'default-tenant',
    '00000000-0000-0000-0000-000000000101',
    'Demo Trading Portfolio',
    100000.00
)
ON CONFLICT (tenant_id, user_id) DO UPDATE
SET name = EXCLUDED.name,
    cash_balance = EXCLUDED.cash_balance,
    updated_at = NOW();

INSERT INTO portfolio.positions (tenant_id, portfolio_id, symbol, quantity, average_price)
VALUES
    ('default-tenant', '00000000-0000-0000-0000-000000000201', 'AAPL', 25, 180.00),
    ('default-tenant', '00000000-0000-0000-0000-000000000201', 'MSFT', 10, 420.00)
ON CONFLICT (portfolio_id, symbol) DO UPDATE
SET quantity = EXCLUDED.quantity,
    average_price = EXCLUDED.average_price,
    updated_at = NOW();
