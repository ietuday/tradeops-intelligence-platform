# Database Migrations

TradeOps v2.4.0 adds a lightweight SQL migration runner for local development, CI validation, and demos.

## Folder Structure

```text
infrastructure/database/
  migrations/
  seeds/
```

Service-owned migrations under `services/*/migrations` still exist. The infrastructure-level migrations provide a repeatable shared PostgreSQL baseline for the platform.

## Naming Convention

Use a zero-padded version prefix and a descriptive name:

```text
007_add_notification_indexes.sql
```

Migrations run in sorted filename order.

## Tracking Table

The runner creates:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    checksum VARCHAR(255),
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Applied migrations are skipped. If a migration file changes after it has been applied, the checksum mismatch causes the runner to fail.

## Run Locally

With Docker Compose PostgreSQL running:

```bash
./scripts/db-migrate.sh
```

Or target a specific database:

```bash
DATABASE_URL=postgres://tradeops:tradeops_dev@localhost:5432/tradeops?sslmode=disable ./scripts/db-migrate.sh
```

## Docker Compose Tool Service

The optional Compose tool service is profile-gated:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml --profile tools run --rm db-migrate
```

Normal `make dev-up` startup is unchanged.

## Verify

```bash
psql "$DATABASE_URL" -c "SELECT version, name, applied_at FROM schema_migrations ORDER BY version;"
```

## Add A Migration

1. Create the next numbered SQL file in `infrastructure/database/migrations`.
2. Use PostgreSQL-compatible SQL.
3. Prefer `CREATE SCHEMA IF NOT EXISTS`, `CREATE TABLE IF NOT EXISTS`, and `CREATE INDEX IF NOT EXISTS`.
4. Avoid destructive SQL such as `DROP`, `TRUNCATE`, or broad `DELETE`.
5. Run `./scripts/db-migrate.sh` twice to confirm idempotency.

## CI Usage

CI validates script syntax and Docker Compose config. Integration-style migration execution can run when PostgreSQL is available:

```bash
./scripts/db-migrate.sh
./scripts/db-migrate.sh
```

## Production Safety Notes

- The runner does not reset databases.
- The runner does not implement rollback logic.
- Checksums protect already-applied migration files from silent edits.
- Production deployments should use controlled credentials and backups before schema changes.
