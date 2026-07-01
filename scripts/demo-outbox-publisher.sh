#!/usr/bin/env bash
set -euo pipefail

ORDER_URL="${ORDER_URL:-http://localhost:8086}"
DATABASE_URL="${ORDER_DATABASE_URL:-postgres://tradeops:tradeops@localhost:5432/tradeops?sslmode=disable}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-60}"
OUTAGE_CHECK="${OUTAGE_CHECK:-false}"

log() { printf '[outbox-demo] %s\n' "$*"; }
fail() { log "FAIL: $*"; exit 1; }

require() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required"
}

require curl
require psql

log "checking order service health at ${ORDER_URL}"
curl -fsS "${ORDER_URL}/health" >/dev/null || fail "order service health check failed"

before_pending="$(psql "${DATABASE_URL}" -Atc "SELECT count(*) FROM order_outbox WHERE published_at IS NULL AND status <> 'failed';")"
log "current unpublished outbox rows: ${before_pending}"

deadline=$((SECONDS + TIMEOUT_SECONDS))
while (( SECONDS < deadline )); do
  pending="$(psql "${DATABASE_URL}" -Atc "SELECT count(*) FROM order_outbox WHERE published_at IS NULL AND status <> 'failed';")"
  failed="$(psql "${DATABASE_URL}" -Atc "SELECT count(*) FROM order_outbox WHERE status = 'failed';")"
  status="$(curl -fsS "${ORDER_URL}/internal/outbox/status" || true)"
  log "pending=${pending} failed=${failed} status=${status}"
  if [[ "${pending}" == "0" ]]; then
    log "PASS: outbox backlog drained"
    break
  fi
  sleep 2
done

pending="$(psql "${DATABASE_URL}" -Atc "SELECT count(*) FROM order_outbox WHERE published_at IS NULL AND status <> 'failed';")"
[[ "${pending}" == "0" ]] || fail "outbox backlog did not drain within ${TIMEOUT_SECONDS}s"

published="$(psql "${DATABASE_URL}" -Atc "SELECT count(*) FROM order_outbox WHERE published_at IS NOT NULL;")"
log "published rows: ${published}"

if [[ "${OUTAGE_CHECK}" == "true" ]]; then
  log "OUTAGE_CHECK=true: stop Redpanda manually, create an order event, then restart Redpanda."
  log "The publisher should increment attempt_count and later publish after recovery."
fi
