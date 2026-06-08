# PostgreSQL Backup And Restore

The local platform uses PostgreSQL in Docker Compose. Backup and restore scripts run PostgreSQL tools inside the Compose `postgres` service.

## Backup

Create a SQL backup:

```bash
./scripts/db-backup.sh
```

Default output:

```text
backups/tradeops_backup_YYYYMMDD_HHMMSS.sql
```

Supported environment variables:

```bash
COMPOSE_FILE=infrastructure/docker/docker-compose.yml
POSTGRES_SERVICE=postgres
POSTGRES_USER=tradeops
POSTGRES_DB=tradeops
BACKUP_DIR=backups
```

## Verify Backup File

```bash
ls -lh backups/
head -n 5 backups/tradeops_backup_YYYYMMDD_HHMMSS.sql
grep -n "PostgreSQL database dump" backups/tradeops_backup_YYYYMMDD_HHMMSS.sql | head
```

## Restore

Restore requires an explicit confirmation flag because it may overwrite local database state:

```bash
./scripts/db-restore.sh backups/tradeops_backup_YYYYMMDD_HHMMSS.sql --confirm
```

Without `--confirm`, the script prints a warning and exits.

## Safety Precautions

- Take a new backup before restoring another backup.
- Stop demo scripts that may write data while restoring.
- Restore only backups you trust.
- Use this for local recovery and demos, not production database administration.

## Manual Commands

Backup through Compose:

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres \
  pg_dump -U tradeops -d tradeops > backups/tradeops_backup_manual.sql
```

Restore through Compose:

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres \
  psql -U tradeops -d tradeops < backups/tradeops_backup_manual.sql
```

