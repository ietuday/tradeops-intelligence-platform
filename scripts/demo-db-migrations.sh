#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-${ROOT_DIR}/infrastructure/docker/.env.example}"

log() {
  printf '[demo-db] %s\n' "$*"
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

ensure_postgres() {
  if psql_cmd -q -c "SELECT 1;" >/dev/null 2>&1; then
    log "PostgreSQL is already reachable."
    return 0
  fi

  if command -v docker >/dev/null 2>&1; then
    log "Starting Docker Compose PostgreSQL..."
    docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" up -d postgres
  else
    log "Docker is not available; db-migrate.sh will wait for PostgreSQL and fail if unreachable."
  fi
}

main() {
  load_env_file "$COMPOSE_ENV_FILE"
  load_env_file "${ROOT_DIR}/infrastructure/docker/.env"
  set_connection_defaults

  log "Running database migration and seed demo."
  ensure_postgres

  "${ROOT_DIR}/scripts/db-migrate.sh"
  "${ROOT_DIR}/scripts/db-seed.sh"

  log "Re-running migrations to prove skip/idempotency behavior."
  "${ROOT_DIR}/scripts/db-migrate.sh"

  log "Re-running seeds to prove idempotency."
  "${ROOT_DIR}/scripts/db-seed.sh"

  log "Applied migrations:"
  psql_cmd -P pager=off -c "SELECT version, name, applied_at FROM schema_migrations ORDER BY version;"

  log "Seed summary:"
  psql_cmd -P pager=off -c "SELECT 'identity.roles' AS table_name, COUNT(*) AS rows FROM identity.roles UNION ALL SELECT 'identity.users', COUNT(*) FROM identity.users UNION ALL SELECT 'portfolio.portfolios', COUNT(*) FROM portfolio.portfolios UNION ALL SELECT 'strategy.strategies', COUNT(*) FROM strategy.strategies;"

  log "Database migration demo complete."
}

main "$@"
