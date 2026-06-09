# Multitenancy Runbook

## User Sees Another Tenant's Data

1. Confirm the JWT contains the expected `tenantId`.
2. Confirm API Gateway forwarded `X-Tenant-ID`.
3. Check the target service query includes `tenant_id`.
4. Inspect audit logs by correlation ID.

## Event Missing tenantId

Older demo events may omit `tenantId`. Consumers default to `default-tenant` for compatibility. New producers should emit top-level `tenantId`.

## WebSocket Client Does Not Receive Events

1. Decode the WebSocket JWT and check `tenantId`.
2. Check the Kafka event payload has the same `tenantId`.
3. Confirm the user role can access the selected stream.
4. Inspect `tradeops_api_gateway_websocket_messages_sent_total`.

## Local Demo Tenant

Demo scripts use:

```bash
TENANT_ID=default-tenant
```

Override it only when testing isolation between tenants.

## Migration Missing

If tenant columns are missing, rerun service migrations or recreate the local Docker volumes for a clean demo database. Do not remove production data without a backup.

