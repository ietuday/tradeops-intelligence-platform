# TradeOps Database Migrations

This directory contains the lightweight SQL-based migration and seed baseline used by `scripts/db-migrate.sh` and `scripts/db-seed.sh`.

## Layout

- `migrations/` contains versioned schema migrations.
- `seeds/` contains idempotent demo seed data.

The service-owned migration folders under `services/*/migrations` remain in place. This infrastructure-level baseline is intended for local setup, CI validation, demos, and interview walkthroughs.

## Run

```bash
./scripts/db-migrate.sh
./scripts/db-seed.sh
```

Set `DATABASE_URL` to target a specific PostgreSQL instance.
