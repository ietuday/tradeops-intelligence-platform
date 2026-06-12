# Admin Console APIs

v2.7.0 adds read-only admin operations APIs inside the API Gateway under `/api/admin`. They aggregate platform health, service metadata, event topics, DLQ guidance, audit activity, surveillance alerts, notifications, rule configuration, and safe runtime configuration for operator workflows.

No frontend UI is included.

## RBAC And Tenant Scope

All `/api/admin` endpoints require a JWT.

Read-only endpoints allow `trading_admin` and `risk_manager`. Mutating actions allow `trading_admin` only. The only mutating endpoint in v2.7.0 is `POST /api/admin/cache/refresh`, which returns `202` with a message when no downstream refresh hook is configured.

Tenant scope comes from the JWT `tenantId`. A `trading_admin` can pass `tenantId` as a query parameter for support workflows; other roles stay scoped to their JWT tenant. The API Gateway forwards the resolved tenant as `X-Tenant-ID` and never adds tenant identifiers as Prometheus labels.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/admin/health-summary` | Bounded health checks across API Gateway and backend services. |
| `GET` | `/api/admin/services` | Static service registry with owners, URLs, health/ready/metrics paths, and topic ownership. |
| `GET` | `/api/admin/topics` | Static event topic catalog aligned with `docs/events/event-catalog.md`. |
| `GET` | `/api/admin/dlq-summary` | Known DLQ topics, owner services, replay script, and runbook links. |
| `GET` | `/api/admin/audit-summary` | Tenant-scoped audit summary from audit-service, with degraded fallback. |
| `GET` | `/api/admin/alerts-summary` | Tenant-scoped surveillance alert summary, with degraded fallback. |
| `GET` | `/api/admin/notifications-summary` | Tenant-scoped notification summary, with degraded fallback. |
| `GET` | `/api/admin/rule-config-summary` | Aggregated surveillance rule config counts. |
| `GET` | `/api/admin/platform-config` | Safe, non-secret runtime configuration summary. |
| `GET` | `/api/admin/ops-checklist` | Links to operational runbooks and checklists. |
| `POST` | `/api/admin/cache/refresh` | Explicit safe admin action; no-op unless refresh hooks are configured. |

## Example

```bash
curl http://localhost:8080/api/admin/health-summary \
  -H "Authorization: Bearer ${TOKEN}"
```

Response:

```json
{
  "status": "HEALTHY",
  "checkedAt": "2026-06-12T10:00:00.000Z",
  "services": [
    {
      "name": "order-service",
      "status": "HEALTHY",
      "healthUrl": "http://order-service:8080/health",
      "latencyMs": 12,
      "error": null
    }
  ]
}
```

Health checks use `ADMIN_HEALTH_TIMEOUT_MS` with a default of `1500` milliseconds and execute concurrently. If all services are healthy, the platform status is `HEALTHY`. If non-critical services fail, the status is `DEGRADED`. If a critical service fails, the status is `UNHEALTHY`.

## Degraded Responses

Summary endpoints call existing downstream APIs where available. If a downstream service returns an error, times out, or is unreachable, the admin endpoint still returns `200` with `status: "DEGRADED"`, an empty summary shape, and an error message. This lets the admin console remain useful during incidents.

## Platform Config Masking

`GET /api/admin/platform-config` reports only safe values. Passwords in URLs are masked, for example:

```text
postgres://tradeops:****@postgres:5432/tradeops
```

JWT secrets, DB passwords, API keys, full secret-bearing connection strings, user IDs, tenant IDs, trace IDs, and correlation IDs are not exposed as metric labels.

## Metrics And Tracing

API Gateway exposes:

| Metric | Purpose |
| --- | --- |
| `tradeops_api_gateway_admin_requests_total{endpoint,status}` | Admin API request count. |
| `tradeops_api_gateway_admin_health_checks_total{service,status}` | Downstream health check outcomes. |
| `tradeops_api_gateway_admin_health_check_duration_ms{service}` | Downstream health check latency. |

Admin routes attach OpenTelemetry span attributes for `admin.endpoint`, `tenant.id`, and `correlation.id` when tracing is enabled. Logs include correlation ID, tenant ID, and user ID when available, without logging secrets.

## Demo

```bash
TOKEN=<jwt> ./scripts/demo-admin-ops.sh
TENANT_ID=default-tenant TOKEN=<jwt> ./scripts/demo-admin-ops.sh
```
