# Platform Operations

The v2.7.0 admin APIs provide a single backend operations layer for local demos and production-readiness discussions. They do not replace service-level runbooks; they aggregate the first set of facts an operator needs before drilling into a service.

## Health Checks

`GET /api/admin/health-summary` checks API Gateway and downstream service health endpoints with short timeouts. The endpoint is intentionally tolerant of partial failures and reports `HEALTHY`, `DEGRADED`, or `UNHEALTHY` based on service outcomes and criticality.

## Service And Topic Visibility

`GET /api/admin/services` exposes the static service registry. `GET /api/admin/topics` exposes known producers, consumers, schema files, and event descriptions. This is static by design in v2.7.0; it avoids adding a Kafka admin dependency only for catalog views.

## DLQ Visibility

`GET /api/admin/dlq-summary` lists `portfolio.dlq`, `surveillance.dlq`, `notification.dlq`, and `audit.dlq`, with the replay helper and DLQ runbook links. Operators should inspect failed payloads and verify idempotency before replaying.

## Activity Summaries

The admin API calls existing downstream summary/list endpoints for audit activity, surveillance alerts, notifications, and rule configuration. When a downstream service is unavailable, the API returns an explicit degraded response instead of failing the entire admin surface.

## Production Considerations

Keep admin APIs behind authentication, least-privilege RBAC, TLS, rate limits, and audit logging. Do not expose secrets through platform config responses. Avoid tenant, user, trace, and correlation identifiers as metric labels. Prefer service-owned mutations and add explicit admin actions only when they are safe, idempotent, and role-protected.

## Limitations

v2.7.0 does not include a frontend UI, live Kafka topic discovery, DLQ message counts, broad cache refresh orchestration, or dangerous actions such as DB resets, data deletion, or DLQ purging.
