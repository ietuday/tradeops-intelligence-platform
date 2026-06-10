INSERT INTO surveillance_rule_configs (
    id,
    tenant_id,
    rule_name,
    enabled,
    severity,
    threshold_numeric,
    threshold_count,
    window_seconds,
    threshold_percent,
    description
) VALUES
    (gen_random_uuid(), 'default-tenant', 'LargeOrderRule', true, 'HIGH', 100000, NULL, NULL, NULL, 'Triggers when order notional exceeds configured threshold'),
    (gen_random_uuid(), 'default-tenant', 'RapidOrderSubmissionRule', true, 'MEDIUM', NULL, 5, 60, NULL, 'Triggers when a user submits too many orders within a rolling window'),
    (gen_random_uuid(), 'default-tenant', 'HighCancelRateRule', true, 'HIGH', NULL, 3, 300, NULL, 'Triggers when a user cancels too many orders within a rolling window'),
    (gen_random_uuid(), 'default-tenant', 'RiskScoreBreachRule', true, 'HIGH', 80, NULL, NULL, NULL, 'Triggers when risk score meets or exceeds configured threshold'),
    (gen_random_uuid(), 'default-tenant', 'AbnormalPriceMovementRule', true, 'MEDIUM', NULL, NULL, NULL, 10, 'Triggers when symbol price moves beyond configured percentage')
ON CONFLICT (tenant_id, rule_name) DO NOTHING;
