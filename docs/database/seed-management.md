# Seed Management

TradeOps seeds are minimal, demo-safe, and idempotent.

## Folder

```text
infrastructure/database/seeds/
```

Seed files run in sorted filename order.

## Run

Run migrations first:

```bash
./scripts/db-migrate.sh
./scripts/db-seed.sh
```

With the Compose tool service:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml --profile tools run --rm db-seed
```

## Idempotency

Seed SQL uses:

```sql
INSERT ... ON CONFLICT DO NOTHING
```

or:

```sql
INSERT ... ON CONFLICT DO UPDATE
```

Repeated runs update deterministic demo rows and do not create duplicates.

## Demo Data

Current seeds include:

- Demo roles.
- Demo users with placeholder password hashes.
- Demo portfolio and positions.
- Demo strategy and strategy rules.

Passwords are placeholders for local/demo use only and are not production credentials.

## Add A Seed

1. Create the next numbered SQL file in `infrastructure/database/seeds`.
2. Keep data small and demo-focused.
3. Use deterministic IDs when rows are referenced by later seed files.
4. Use conflict handling on natural keys or unique constraints.
5. Run `./scripts/db-seed.sh` twice.

## Safety Notes

- Seeds must not delete or truncate data.
- Do not include real users, passwords, API keys, or customer data.
- Production seeding should be reviewed separately from local demo seeding.
