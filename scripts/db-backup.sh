#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
POSTGRES_SERVICE="${POSTGRES_SERVICE:-postgres}"
POSTGRES_USER="${POSTGRES_USER:-tradeops}"
POSTGRES_DB="${POSTGRES_DB:-tradeops}"
BACKUP_DIR="${BACKUP_DIR:-${ROOT_DIR}/backups}"

timestamp="$(date +%Y%m%d_%H%M%S)"
backup_file="${BACKUP_DIR}/tradeops_backup_${timestamp}.sql"

echo "Creating PostgreSQL backup..."
echo "Compose file: ${COMPOSE_FILE}"
echo "Postgres service: ${POSTGRES_SERVICE}"
echo "Database: ${POSTGRES_DB}"

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is required for backup." >&2
  exit 1
fi

mkdir -p "${BACKUP_DIR}"

if docker compose -f "${COMPOSE_FILE}" exec -T "${POSTGRES_SERVICE}" pg_dump -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" > "${backup_file}"; then
  echo "Backup created: ${backup_file}"
else
  rm -f "${backup_file}"
  echo "ERROR: backup failed. Is the Compose postgres service running?" >&2
  exit 1
fi

if [ ! -s "${backup_file}" ]; then
  echo "ERROR: backup file is empty: ${backup_file}" >&2
  exit 1
fi

echo "Backup size:"
ls -lh "${backup_file}"

