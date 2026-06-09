#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${DB_MIGRATIONS_DIR:-${ROOT_DIR}/infrastructure/database/migrations}"

log() {
  printf '[db-migrate] %s\n' "$*"
}

error() {
  printf '[db-migrate] ERROR: %s\n' "$*" >&2
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

ensure_tracking_table() {
  psql_cmd -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    checksum VARCHAR(255),
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL
}

checksum_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    cksum "$file" | awk '{print $1}'
  fi
}

sql_escape() {
  printf "%s" "$1" | sed "s/'/''/g"
}

applied_checksum() {
  local version="$1"
  psql_cmd -v ON_ERROR_STOP=1 -tA -c "SELECT COALESCE(checksum, '') FROM schema_migrations WHERE version = '$(sql_escape "$version")';"
}

record_migration() {
  local version="$1"
  local name="$2"
  local checksum="$3"
  psql_cmd -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations (version, name, checksum) VALUES ('$(sql_escape "$version")', '$(sql_escape "$name")', '$(sql_escape "$checksum")');" >/dev/null
}

run_migration() {
  local file="$1"
  local base version name checksum existing

  base="$(basename "$file")"
  version="${base%%_*}"
  name="${base#${version}_}"
  name="${name%.sql}"
  checksum="$(checksum_file "$file")"
  existing="$(applied_checksum "$version")"

  if [[ -n "$existing" ]]; then
    if [[ "$existing" != "$checksum" ]]; then
      error "Checksum mismatch for already-applied migration ${version} (${base})."
      error "Recorded: ${existing}"
      error "Current:  ${checksum}"
      return 1
    fi
    log "Skipping already-applied migration ${base}."
    return 0
  fi

  log "Applying migration ${base}."
  psql_cmd -v ON_ERROR_STOP=1 -f "$file" >/dev/null
  record_migration "$version" "$name" "$checksum"
  log "Applied migration ${base}."
}

main() {
  load_env_defaults
  set_connection_defaults

  if [[ ! -d "$MIGRATIONS_DIR" ]]; then
    error "Migrations directory not found: ${MIGRATIONS_DIR}"
    exit 1
  fi

  wait_for_db
  ensure_tracking_table

  local found=false
  while IFS= read -r file; do
    found=true
    run_migration "$file"
  done < <(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name '*.sql' | sort)

  if [[ "$found" == "false" ]]; then
    log "No migration files found in ${MIGRATIONS_DIR}."
  fi

  log "Database migrations complete."
}

main "$@"
