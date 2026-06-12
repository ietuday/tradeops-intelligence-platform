# Surveillance Rule Configuration

v2.6.0 adds database-backed, tenant-aware configuration for surveillance rules. Existing environment variables remain the fallback, so alert generation continues to work if the rule config table is empty or temporarily unavailable.

## Default Rules

| Rule | Default severity | Default threshold |
| --- | --- | --- |
| `LargeOrderRule` | `HIGH` | `thresholdNumeric=100000` |
| `RapidOrderSubmissionRule` | `MEDIUM` | `thresholdCount=5`, `windowSeconds=60` |
| `HighCancelRateRule` | `HIGH` | `thresholdCount=3`, `windowSeconds=300` |
| `RiskScoreBreachRule` | `HIGH` | `thresholdNumeric=80` |
| `AbnormalPriceMovementRule` | `MEDIUM` | `thresholdPercent=10` |

## Storage

Rule configs are stored in `surveillance_rule_configs` with `tenant_id`, `rule_name`, `enabled`, severity, threshold fields, optional JSON config, description, updater, and timestamps.

The service creates default rows for `default-tenant` on startup using `ON CONFLICT DO NOTHING`. The central seed file also inserts the same defaults for local database demos without overwriting customized values.

## Fallback Order

Effective config resolution is:

1. Tenant-specific database config.
2. `default-tenant` database config.
3. Environment/default hardcoded config.

The rule engine caches effective configs in memory and refreshes the tenant cache after API updates. If DB loading fails, existing environment defaults remain active.

## APIs

Base path:

```text
/api/v1/surveillance/rules
```

Through API Gateway:

```text
/api/surveillance/rules
```

Admin summary:

```text
/api/admin/rule-config-summary
```

The admin endpoint aggregates total/enabled/disabled/severity counts from the surveillance rule API and degrades gracefully if surveillance-service is unavailable.

List rules:

```bash
curl "http://localhost:8080/api/surveillance/rules" \
  -H "Authorization: Bearer <jwt>" \
  -H "X-Tenant-ID: default-tenant"
```

Update a rule:

```bash
curl -X PUT "http://localhost:8080/api/surveillance/rules/LargeOrderRule" \
  -H "Authorization: Bearer <jwt>" \
  -H "X-Tenant-ID: default-tenant" \
  -H "Content-Type: application/json" \
  -d '{"enabled":true,"severity":"CRITICAL","thresholdNumeric":250000}'
```

Enable or disable:

```bash
curl -X POST "http://localhost:8080/api/surveillance/rules/LargeOrderRule/disable" \
  -H "Authorization: Bearer <jwt>" \
  -H "X-Tenant-ID: default-tenant"

curl -X POST "http://localhost:8080/api/surveillance/rules/LargeOrderRule/enable" \
  -H "Authorization: Bearer <jwt>" \
  -H "X-Tenant-ID: default-tenant"
```

## Dry-Run Simulation

Use rule simulation before applying threshold changes:

```bash
curl -X POST "http://localhost:8080/api/surveillance/rules/LargeOrderRule/simulate" \
  -H "Authorization: Bearer <jwt>" \
  -H "X-Tenant-ID: default-tenant" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"default-tenant","lookbackMinutes":60,"dryRun":true,"config":{"thresholdNumeric":200000}}'
```

The simulator reads the tenant-effective config, overlays proposed values in memory, and returns would-trigger alert counts without updating `surveillance_rule_configs`, refreshing the live cache, creating alerts, or publishing `surveillance.alert.*` events. See [rule simulation](rule-simulation.md).

## RBAC

Read access:

- `trading_admin`
- `risk_manager`
- `analyst`
- `viewer`

Write, enable, and disable access:

- `trading_admin`
- `risk_manager`

## Events

Rule changes publish Kafka/Redpanda events:

- `surveillance.rule_config.updated`
- `surveillance.rule_config.enabled`
- `surveillance.rule_config.disabled`
- `surveillance.rule_simulation.requested`
- `surveillance.rule_simulation.completed`
- `surveillance.rule_simulation.failed`

The event schemas live under `schemas/events/surveillance/`. Audit-service can consume and normalize these events in a future pass; for now the events are available for compliance and debugging integrations.

## Metrics

- `surveillance_rule_config_updates_total{rule_name,action}`
- `surveillance_rule_config_reload_total{status}`
- `surveillance_rule_disabled_skips_total{rule_name}`
- `surveillance_rule_config_cache_entries`
- `surveillance_rule_simulation_requests_total{rule_name,status}`
- `surveillance_rule_simulation_duration_seconds_bucket{rule_name,status}`
- `surveillance_rule_simulation_matches_total{rule_name}`
- `surveillance_rule_simulation_failures_total{rule_name}`

Tenant IDs are intentionally not metric labels.

## Demo

Dry-run:

```bash
TOKEN=<jwt> ./scripts/demo-rule-config.sh
```

Apply a temporary LargeOrderRule threshold and restore it:

```bash
TOKEN=<jwt> ./scripts/demo-rule-config.sh --apply
```

Rule simulation:

```bash
TOKEN=<jwt> ./scripts/demo-rule-simulation.sh
```

## Known Limitations

- No custom user-authored rules yet.
- Rule names are fixed and validated.
- Cache refresh is update-driven rather than a distributed invalidation mechanism.
- Rule config change events are published, but audit normalization is documented as a future extension.
- Rule simulation currently uses deterministic demo/historical-style events rather than a production event warehouse.
