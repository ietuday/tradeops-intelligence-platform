# Database Migration Runbook

Use this runbook when migrations or seeds fail locally or in CI.

## PostgreSQL Not Reachable

Check Compose status:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml ps postgres
```

Start PostgreSQL:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml up -d postgres
```

Then rerun:

```bash
./scripts/db-migrate.sh
```

## psql Not Installed

The host scripts require `psql`. If it is not installed, use the Compose tool service:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml --profile tools run --rm db-migrate
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml --profile tools run --rm db-seed
```

## Checksum Mismatch

Symptom: `Checksum mismatch for already-applied migration`.

Cause: a migration file changed after being recorded in `schema_migrations`.

Fix: Do not edit applied migrations. Add a new migration with the next version number.

## Seed Fails

1. Confirm migrations have run.
2. Check the failing seed file for missing referenced rows.
3. Confirm `ON CONFLICT` targets match a unique constraint.
4. Rerun `./scripts/db-seed.sh`.

## Verify Applied Migrations

```bash
psql "$DATABASE_URL" -c "SELECT version, name, applied_at FROM schema_migrations ORDER BY version;"
```

## Verify Seeded Rows

```bash
psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM identity.roles;"
psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM identity.users;"
```

## Demo

```bash
./scripts/demo-db-migrations.sh
```

The demo runs migrations, runs seeds, repeats both commands, prints `schema_migrations`, and prints a small seed summary.
