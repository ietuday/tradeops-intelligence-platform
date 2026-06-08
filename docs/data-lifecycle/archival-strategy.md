# Archival Strategy

Archival in this project means exporting old operational rows to local files before any cleanup is considered. It is intentionally simple: no object storage, no warehouse, and no new archival service.

## Archive-Before-Delete

The platform follows an archive-before-delete approach:

1. Identify candidate rows using documented retention windows.
2. Export candidates to CSV files.
3. Validate row counts and file contents.
4. Keep data in PostgreSQL unless an explicit destructive confirmation flag is used.
5. Restore from a database backup if cleanup produces an unexpected result.

## Local Archive Directory

Use date-partitioned local directories:

```text
archives/YYYY-MM-DD/
```

Example:

```text
archives/2026-06-08/market_ticks.csv
archives/2026-06-08/notifications.csv
```

The archive directory should not be committed to git.

## Export Format

The default archival script exports CSV with headers using PostgreSQL `\copy` through Docker Compose. CSV is easy to inspect locally and easy to load into another database if needed.

JSON exports can be added later for event-shaped records, but CSV is the simplest operational default for table archival.

## Why Deletes Are Not Automatic

Deletion can remove audit context, break demo repeatability, and make incident analysis harder. For that reason:

- `scripts/archive-old-data.sh` runs in dry-run/export mode by default.
- Deletes require `--delete-confirm`.
- Orders, surveillance alerts, and audit logs are documented retention domains but are not deleted by the default script.

## Validate Archive Output

After exporting:

```bash
ls -lh archives/YYYY-MM-DD/
head -n 5 archives/YYYY-MM-DD/market_ticks.csv
wc -l archives/YYYY-MM-DD/market_ticks.csv
```

Compare the script row counts with the exported file line count. CSV files include a header row, so the line count should be row count plus one when rows were exported.

## Restore Safety

Take a database backup before destructive cleanup:

```bash
./scripts/db-backup.sh
```

If cleanup causes a local issue, restore from the backup:

```bash
./scripts/db-restore.sh backups/tradeops_backup_YYYYMMDD_HHMMSS.sql --confirm
```

## Future Improvements

- Store archives in object storage or a managed warehouse.
- Add checksums and archive manifests.
- Add schema-aware JSON export for event records.
- Add retention jobs in production orchestration only after policies are approved.

