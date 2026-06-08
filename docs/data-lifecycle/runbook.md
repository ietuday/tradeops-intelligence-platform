# Data Lifecycle Runbook

Use these runbooks for local operations. Commands are designed for Docker Compose and local demo data.

## Take A Database Backup

Use case: Preserve local state before cleanup, restore testing, or a risky demo.

Command:

```bash
./scripts/db-backup.sh
```

Expected output: A file path such as `backups/tradeops_backup_20260608_120000.sql`.

Safety notes: Verify the file exists before running archive or restore operations.

## Restore A Database Backup

Use case: Recover local state from a known backup.

Command:

```bash
./scripts/db-restore.sh backups/tradeops_backup_20260608_120000.sql --confirm
```

Expected output: Restore progress and a completion message.

Safety notes: This may overwrite local database state. Take a fresh backup first when possible.

## Archive Old Data

Use case: Export old high-volume records without deleting them.

Command:

```bash
./scripts/archive-old-data.sh
```

Expected output: Row counts and CSV files under `archives/YYYY-MM-DD/`.

Safety notes: Dry-run/export mode is the default. Deletion requires `--delete-confirm`.

## Delete Archived Candidate Rows

Use case: Local cleanup after validating archive output and taking a backup.

Command:

```bash
./scripts/archive-old-data.sh --delete-confirm
```

Expected output: CSV exports followed by delete counts for the same retention predicates.

Safety notes: Orders, surveillance alerts, and audit logs are not deleted by this default script.

## Replay Sample Events

Use case: Generate known event traffic for demos or event-consumer checks.

Command:

```bash
./scripts/replay-sample-events.sh --all
```

Expected output: Published sample payloads or manual publish commands if Docker/Redpanda is unavailable.

Safety notes: Replay can create duplicate records or metrics; use known-good sample payloads.

## Replay DLQ Events

Use case: Investigate a failed event and prepare a controlled replay.

Command:

```bash
./scripts/replay-dlq-events.sh --topic surveillance.dlq --dry-run
```

Expected output: Inspection and manual replay guidance.

Safety notes: Fix the root cause before replay. The script does not bulk replay DLQ messages automatically.

## Validate Archive Output

Use case: Confirm archive files were written correctly.

Command:

```bash
ls -lh archives/$(date +%F)/
head -n 5 archives/$(date +%F)/market_ticks.csv
wc -l archives/$(date +%F)/*.csv
```

Expected output: CSV files with headers and line counts matching exported rows plus one header row.

Safety notes: Archive files may contain sensitive data and should not be committed.

## Investigate Data Growth

Use case: Understand which tables are growing.

Command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres \
  psql -U tradeops -d tradeops -c "SELECT relname, n_live_tup FROM pg_stat_user_tables ORDER BY n_live_tup DESC;"
```

Expected output: Table names ordered by approximate live row count.

Safety notes: `pg_stat_user_tables` counts are approximate but good enough for local investigation.

## Clean Local Demo Environment Safely

Use case: Reset local demos without surprise data loss.

Command:

```bash
./scripts/db-backup.sh
./scripts/archive-old-data.sh
make dev-down
```

Expected output: Backup and archive files before the stack is stopped.

Safety notes: Do not remove Docker volumes unless you intentionally want to lose local database state.

