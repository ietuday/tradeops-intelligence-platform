#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
POSTGRES_SERVICE="${POSTGRES_SERVICE:-postgres}"
POSTGRES_USER="${POSTGRES_USER:-tradeops}"
POSTGRES_DB="${POSTGRES_DB:-tradeops}"

usage() {
  cat <<EOF
Usage:
  ./scripts/db-restore.sh <backup-file.sql> --confirm

This may overwrite local database state.
EOF
}

backup_file="${1:-}"
confirm="${2:-}"

if [ -z "${backup_file}" ]; then
  usage
  exit 1
fi

if [ "${confirm}" != "--confirm" ]; then
  echo "WARNING: This may overwrite local database state."
  echo "Restore is intentionally blocked unless --confirm is passed."
  usage
  exit 1
fi

if [ ! -f "${backup_file}" ]; then
  echo "ERROR: backup file does not exist: ${backup_file}" >&2
  exit 1
fi

if [ ! -s "${backup_file}" ]; then
  echo "ERROR: backup file is empty: ${backup_file}" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is required for restore." >&2
  exit 1
fi

echo "Restoring PostgreSQL backup..."
echo "Compose file: ${COMPOSE_FILE}"
echo "Postgres service: ${POSTGRES_SERVICE}"
echo "Database: ${POSTGRES_DB}"
echo "Backup file: ${backup_file}"
echo "WARNING: This may overwrite local database state."

docker compose -f "${COMPOSE_FILE}" exec -T "${POSTGRES_SERVICE}" psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" < "${backup_file}"

echo "Restore complete."

