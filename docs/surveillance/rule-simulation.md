# Surveillance Rule Simulation

v2.8.0 adds a dry-run simulation path for testing proposed surveillance rule config changes before applying them live.

Simulation is read-only with respect to live rule configuration and alerts:

- It reads tenant-effective rule configs.
- It overlays proposed config values in memory for the simulation request only.
- It evaluates generated demo/historical-style events with a throwaway rule engine.
- It returns matched events, would-trigger alert counts, and a small set of sample matches.
- It does not update `surveillance_rule_configs`.
- It does not refresh or mutate the live rule config cache.
- It does not create live alerts.
- It does not publish `surveillance.alert.*` events or trigger notifications.

## APIs

Base service path:

```text
/api/v1/surveillance/rules
```

Through API Gateway:

```text
/api/surveillance/rules
```

Single-rule simulation:

```bash
curl -X POST "http://localhost:8080/api/surveillance/rules/LargeOrderRule/simulate" \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  --data '{"tenantId":"default-tenant","lookbackMinutes":60,"dryRun":true,"config":{"thresholdNumeric":200000}}'
```

Bulk simulation:

```bash
curl -X POST "http://localhost:8080/api/surveillance/rules/simulate" \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  --data '{"tenantId":"default-tenant","lookbackMinutes":60,"dryRun":true,"rules":[{"ruleName":"LargeOrderRule","config":{"thresholdNumeric":200000}},{"ruleName":"price_spike","config":{"threshold":5}}]}'
```

## Request Fields

| Field | Required | Notes |
| --- | --- | --- |
| `tenantId` | Yes | Must match the caller tenant unless the caller has `trading_admin`. |
| `ruleName` | Yes for single-rule body when no path rule is present | Canonical rule names are preferred; aliases such as `price_spike` are accepted for simulation. |
| `config` | No | Proposed values only for the dry run. |
| `lookbackMinutes` | No | Defaults to `60`; maximum is `1440`. |
| `dryRun` | No | Defaults to true. `false` is rejected. |
| `rules` | Bulk only | One or more rules to simulate. |

Supported config fields mirror rule configuration APIs: `enabled`, `severity`, `thresholdNumeric`, `thresholdCount`, `thresholdPercent`, `windowSeconds`, `config`, and `description`. The generic `threshold` field is accepted for demo-friendly simulation requests and mapped to the relevant rule threshold type.

## Response

```json
{
  "tenantId": "default-tenant",
  "ruleName": "LargeOrderRule",
  "dryRun": true,
  "lookbackMinutes": 60,
  "matchedEvents": 1,
  "wouldTriggerAlerts": 1,
  "sampleMatches": [
    {
      "symbol": "NVDA",
      "observedValue": 285000,
      "threshold": 200000,
      "timestamp": "2026-06-10T10:30:00Z",
      "reason": "Order notional 285000.00 exceeds configured threshold 200000.00"
    }
  ]
}
```

## Events

Simulation publishes lifecycle events only:

- `surveillance.rule_simulation.requested`
- `surveillance.rule_simulation.completed`
- `surveillance.rule_simulation.failed`

These are not alert events and should not be wired to notification flows.

## Metrics

- `surveillance_rule_simulation_requests_total{rule_name,status}`
- `surveillance_rule_simulation_duration_seconds_bucket{rule_name,status}`
- `surveillance_rule_simulation_matches_total{rule_name}`
- `surveillance_rule_simulation_failures_total{rule_name}`

Tenant IDs are intentionally not Prometheus labels; use correlation IDs and lifecycle events for per-run debugging.

## Demo

```bash
TOKEN=<jwt> ./scripts/demo-rule-simulation.sh
```

The script fetches the live config, runs current/strict/relaxed threshold simulations, compares would-trigger alert counts, and confirms the live config remains unchanged.
