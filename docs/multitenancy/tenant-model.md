# Tenant Model

TradeOps uses a shared database multitenancy model. Tenant-owned tables carry a `tenant_id` column and application queries filter by the current tenant.

## Defaults

- Default tenant ID: `default-tenant`
- Stable default tenant UUID for identity records: `00000000-0000-0000-0000-000000000001`
- Standard JWT claim: `tenantId`
- Standard HTTP propagation header: `X-Tenant-ID`
- Standard Kafka/Redpanda event field: `tenantId`

Existing local demo data is backfilled to the default tenant so older demos continue to work.

v2.5.0 event schemas keep `tenantId` optional for backward compatibility with older demo payloads, but tenant-owned producers should include it in new events. The event envelope standard also recommends `eventVersion`, `correlationId`, and `traceparent` metadata where available.

## Why Shared Database

This portfolio project uses shared PostgreSQL tables with tenant filters because it is simple to run locally, easy to explain in interviews, and avoids the operational overhead of database-per-tenant provisioning. The model still demonstrates the important backend concepts: identity-scoped access, query isolation, event propagation, audit trails, and real-time stream filtering.

Database-per-tenant or schema-per-tenant can be introduced later for stronger isolation, enterprise contracts, or compliance boundaries.

## Tenant Sources

External clients normally rely on the `tenantId` JWT claim. API Gateway forwards that value to downstream services as `X-Tenant-ID`.

`X-Tenant-ID` from external clients is ignored when a JWT has a tenant unless the caller has `trading_admin`. Local demos may use the header when no JWT tenant is available.

## Owned Data

Tenant-owned data includes:

- users
- orders and order events
- portfolios, holdings, snapshots, and processed events
- surveillance alerts and rule executions
- notifications, delivery attempts, and preferences
- audit logs and export requests
- strategy and risk records where supported

Market data remains global by default. Tenant-specific market feeds are documented as a future improvement.
