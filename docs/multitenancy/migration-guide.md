# Multitenancy Migration Guide

v2.2.0 introduces tenant-aware storage with additive migrations.

## Strategy

1. Add nullable `tenant_id` columns.
2. Backfill existing rows to `default-tenant`.
3. Add tenant indexes.
4. Enforce tenant context in application reads and writes.
5. Defer `NOT NULL` constraints until production data quality is verified.

This keeps existing local demo data safe and avoids a hard migration failure on older development databases.

## Default Tenant

Identity creates a stable default tenant:

```text
id: 00000000-0000-0000-0000-000000000001
slug: default
name: Default Tenant
```

Other services use `default-tenant` as their application-level tenant ID for existing rows and demo payloads.

## Local Validation

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml config
go test ./... ./services/order-service
```

Run service-specific tests from the service directory when making code changes.

## Future Hardening

- Convert `tenant_id` columns to `NOT NULL`.
- Add tenant-specific partitions.
- Add tenant-specific encryption keys.
- Add tenant-specific rate limits.
- Consider schema-per-tenant or database-per-tenant for enterprise isolation.

