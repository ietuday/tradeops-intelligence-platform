#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SEEDS_DIR="${DB_SEEDS_DIR:-${ROOT_DIR}/infrastructure/database/seeds}"

log() {
  printf '[db-seed] %s\n' "$*"
}

error() {
  printf '[db-seed] ERROR: %s\n' "$*" >&2
}

load_env_defaults() {
  load_env_file "${ROOT_DIR}/infrastructure/docker/.env.example"
  load_env_file "${ROOT_DIR}/infrastructure/docker/.env"
}

load_env_file() {
  local file="$1"
  [[ -f "$file" ]] || return 0

  local line key value
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" == *"="* ]] || continue
    key="${line%%=*}"
    value="${line#*=}"
    key="$(printf '%s' "$key" | xargs)"
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    if [[ -z "${!key:-}" ]]; then
      export "$key=$value"
    fi
  done < "$file"
}

set_connection_defaults() {
  POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
  POSTGRES_PORT="${POSTGRES_PORT:-5432}"
  POSTGRES_DB="${POSTGRES_DB:-tradeops}"
  POSTGRES_USER="${POSTGRES_USER:-tradeops}"
  POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-tradeops_dev}"
  export POSTGRES_HOST POSTGRES_PORT POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD
}

psql_cmd() {
  if [[ -n "${DATABASE_URL:-}" ]]; then
    psql "$DATABASE_URL" "$@"
  else
    PGPASSWORD="$POSTGRES_PASSWORD" psql \
      -h "$POSTGRES_HOST" \
      -p "$POSTGRES_PORT" \
      -U "$POSTGRES_USER" \
      -d "$POSTGRES_DB" \
      "$@"
  fi
}

wait_for_db() {
  local attempts="${DB_WAIT_ATTEMPTS:-30}"
  local delay="${DB_WAIT_SECONDS:-2}"

  log "Waiting for PostgreSQL..."
  for ((attempt = 1; attempt <= attempts; attempt++)); do
    if psql_cmd -v ON_ERROR_STOP=1 -q -c "SELECT 1;" >/dev/null 2>&1; then
      log "PostgreSQL is reachable."
      return 0
    fi
    log "PostgreSQL not ready yet (${attempt}/${attempts}); retrying in ${delay}s."
    sleep "$delay"
  done

  error "PostgreSQL was not reachable after ${attempts} attempts."
  return 1
}

ensure_migrations_applied() {
  if ! psql_cmd -v ON_ERROR_STOP=1 -tA -c "SELECT 1 FROM schema_migrations LIMIT 1;" >/dev/null 2>&1; then
    error "schema_migrations not found or no migrations applied. Run ./scripts/db-migrate.sh first."
    return 1
  fi
}

run_seed() {
  local file="$1"
  log "Applying seed $(basename "$file")."
  psql_cmd -v ON_ERROR_STOP=1 -f "$file" >/dev/null
}

main() {
  load_env_defaults
  set_connection_defaults

  if [[ ! -d "$SEEDS_DIR" ]]; then
    error "Seeds directory not found: ${SEEDS_DIR}"
    exit 1
  fi

  wait_for_db
  ensure_migrations_applied

  local found=false
  while IFS= read -r file; do
    found=true
    run_seed "$file"
  done < <(find "$SEEDS_DIR" -maxdepth 1 -type f -name '*.sql' | sort)

  if [[ "$found" == "false" ]]; then
    log "No seed files found in ${SEEDS_DIR}."
  fi

  log "Database seeds complete."
}

main "$@"
