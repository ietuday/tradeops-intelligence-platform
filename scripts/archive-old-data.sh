#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
POSTGRES_SERVICE="${POSTGRES_SERVICE:-postgres}"
POSTGRES_USER="${POSTGRES_USER:-tradeops}"
POSTGRES_DB="${POSTGRES_DB:-tradeops}"
ARCHIVE_DIR="${ARCHIVE_DIR:-${ROOT_DIR}/archives}"
RETENTION_DAYS_MARKET_TICKS="${RETENTION_DAYS_MARKET_TICKS:-30}"
RETENTION_DAYS_NOTIFICATIONS="${RETENTION_DAYS_NOTIFICATIONS:-90}"
RETENTION_DAYS_AUDIT_EXPORTS="${RETENTION_DAYS_AUDIT_EXPORTS:-90}"
RETENTION_DAYS_PORTFOLIO_SNAPSHOTS="${RETENTION_DAYS_PORTFOLIO_SNAPSHOTS:-180}"
RETENTION_DAYS_RISK_SCORES="${RETENTION_DAYS_RISK_SCORES:-180}"

delete_confirm="false"
if [ "${1:-}" = "--delete-confirm" ]; then
  delete_confirm="true"
elif [ "${1:-}" != "" ]; then
  echo "Usage: ./scripts/archive-old-data.sh [--delete-confirm]" >&2
  exit 1
fi

archive_date="$(date +%F)"
output_dir="${ARCHIVE_DIR}/${archive_date}"

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is required for archive export." >&2
  exit 1
fi

mkdir -p "${output_dir}"

psql_exec() {
  docker compose -f "${COMPOSE_FILE}" exec -T "${POSTGRES_SERVICE}" psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" "$@"
}

table_exists() {
  local table="$1"
  local exists
  exists="$(psql_exec -Atc "SELECT to_regclass('public.${table}') IS NOT NULL;")"
  [ "${exists}" = "t" ]
}

archive_table() {
  local table="$1"
  local column="$2"
  local days="$3"
  local file="${output_dir}/${table}.csv"
  local predicate="${column} < NOW() - INTERVAL '${days} days'"
  local count

  if ! table_exists "${table}"; then
    echo "SKIP: ${table} does not exist"
    return 0
  fi

  count="$(psql_exec -Atc "SELECT COUNT(*) FROM ${table} WHERE ${predicate};")"
  echo "Table ${table}: ${count} candidate rows older than ${days} days"

  psql_exec -c "\\copy (SELECT * FROM ${table} WHERE ${predicate} ORDER BY ${column}) TO STDOUT WITH CSV HEADER" > "${file}"
  echo "Archive output: ${file}"

  if [ "${delete_confirm}" = "true" ]; then
    local delete_count
    delete_count="$(psql_exec -Atc "SELECT COUNT(*) FROM ${table} WHERE ${predicate};")"
    if [ "${delete_count}" != "${count}" ]; then
      echo "SKIP DELETE: ${table} candidate count changed after export (${count} -> ${delete_count}). Re-run archive before deleting." >&2
      return 0
    fi
    echo "Deleting archived candidates from ${table}..."
    psql_exec -c "DELETE FROM ${table} WHERE ${predicate};"
  else
    echo "Dry-run mode: no rows deleted from ${table}"
  fi
}

echo "TradeOps old-data archive"
echo "Archive directory: ${output_dir}"
if [ "${delete_confirm}" = "true" ]; then
  echo "DELETE CONFIRMED: matching archived candidates will be deleted."
else
  echo "Dry-run/export mode: no rows will be deleted."
fi

archive_table "market_ticks" "received_at" "${RETENTION_DAYS_MARKET_TICKS}"
archive_table "portfolio_snapshots" "created_at" "${RETENTION_DAYS_PORTFOLIO_SNAPSHOTS}"
archive_table "risk_scores" "created_at" "${RETENTION_DAYS_RISK_SCORES}"
archive_table "notifications" "created_at" "${RETENTION_DAYS_NOTIFICATIONS}"
archive_table "notification_delivery_attempts" "attempted_at" "${RETENTION_DAYS_NOTIFICATIONS}"
archive_table "audit_export_requests" "created_at" "${RETENTION_DAYS_AUDIT_EXPORTS}"

echo "Archive complete: ${output_dir}"
echo "Orders, surveillance_alerts, and audit_logs are intentionally not deleted by this script."
