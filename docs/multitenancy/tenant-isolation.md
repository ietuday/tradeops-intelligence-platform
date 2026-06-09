# Tenant Isolation

Tenant isolation is enforced at the application and query layer.

## HTTP APIs

Services resolve tenant context in this order:

1. JWT `tenantId` claim.
2. `X-Tenant-ID` propagated by API Gateway.
3. `default-tenant` fallback for local demo compatibility.

Tenant-owned list, get, update, and lifecycle APIs include `tenant_id` filters. Admin users can still use platform-wide views only where explicitly supported.

## Events

Kafka/Redpanda events include a top-level `tenantId` field when the event belongs to a tenant. Consumers preserve the tenant from incoming events and default to `default-tenant` only when older demo payloads omit it.

## Audit

Audit records store `tenant_id` and exports include tenant information. Audit queries are tenant-scoped by default.

## WebSocket Streams

WebSocket clients inherit tenant context from the JWT used during upgrade. Events with a `tenantId` are delivered only to clients in the same tenant or to `trading_admin` clients.

Events without `tenantId` are treated as global compatibility events and may be delivered to authorized stream subscribers. New tenant-owned events should always include `tenantId`.

## Metrics

Tenant IDs are intentionally not used as Prometheus labels. Tenant labels can create high-cardinality metrics and degrade observability systems. Use logs and audit trails for tenant-level investigation.

