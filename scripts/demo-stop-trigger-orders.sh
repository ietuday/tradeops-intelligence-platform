#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
ORDER_URL="${ORDER_URL:-http://localhost:8086}"
DATABASE_URL="${ORDER_DATABASE_URL:-postgres://tradeops:tradeops@localhost:5432/tradeops?sslmode=disable}"
TENANT_ID="${TENANT_ID:-default-tenant}"
USER_ID="${USER_ID:-demo-stop-user}"
SYMBOL="${SYMBOL:-AAPL}"
STOP_PRICE="${STOP_PRICE:-110.00}"
LOW_PRICE="${LOW_PRICE:-109.00}"
HIGH_PRICE="${HIGH_PRICE:-111.20}"
TOPIC="${STOP_TRIGGER_MARKET_TOPIC:-market.ticks}"
CORRELATION_ID="${CORRELATION_ID:-demo-stop-trigger-$(date +%s)}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-45}"

log() { printf '[stop-trigger-demo] %s\n' "$*"; }
fail() { log "FAIL: $*"; exit 1; }
require() { command -v "$1" >/dev/null 2>&1 || fail "$1 is required"; }

require curl
require psql
require docker

publish_price() {
  local price="$1"
  local event_time
  event_time="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  printf '{"eventType":"market.price.updated","eventVersion":"v1","tenantId":"%s","symbol":"%s","price":"%s","source":"demo","eventTime":"%s","correlationId":"%s"}\n' \
    "${TENANT_ID}" "${SYMBOL}" "${price}" "${event_time}" "${CORRELATION_ID}" |
    docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${TOPIC}" >/dev/null
}

wait_for_sql() {
  local description="$1"
  local sql="$2"
  local expected="$3"
  local deadline=$((SECONDS + TIMEOUT_SECONDS))
  while (( SECONDS < deadline )); do
    value="$(psql "${DATABASE_URL}" -Atc "${sql}")"
    if [[ "${value}" == "${expected}" ]]; then
      log "OK: ${description}"
      return 0
    fi
    sleep 1
  done
  fail "timed out waiting for ${description}"
}

log "checking order service at ${ORDER_URL}"
curl -fsS "${ORDER_URL}/health" >/dev/null || fail "order service health check failed"

log "creating BUY STOP order for ${SYMBOL} stop=${STOP_PRICE}"
ORDER_ID="$(psql "${DATABASE_URL}" -Atc "
INSERT INTO orders (
  tenant_id, user_id, symbol, side, order_type, quantity, filled_quantity, remaining_quantity,
  stop_price, time_in_force, version, risk_status, status, correlation_id
) VALUES (
  '${TENANT_ID}', '${USER_ID}', '${SYMBOL}', 'BUY', 'STOP', 10, 0, 10,
  ${STOP_PRICE}, 'GTC', 1, 'APPROVED', 'accepted', '${CORRELATION_ID}'
) RETURNING id;
")"
log "order_id=${ORDER_ID}"

log "publishing below-stop reference price ${LOW_PRICE}"
publish_price "${LOW_PRICE}"
wait_for_sql "reference price stored below stop" "SELECT price::text FROM reference_prices WHERE tenant_id='${TENANT_ID}' AND symbol='${SYMBOL}'" "${LOW_PRICE}"
sleep 2
current_type="$(psql "${DATABASE_URL}" -Atc "SELECT order_type FROM orders WHERE id='${ORDER_ID}'")"
[[ "${current_type}" == "STOP" ]] || fail "order triggered too early: order_type=${current_type}"
log "OK: order did not trigger below stop"

log "publishing crossing reference price ${HIGH_PRICE}"
publish_price "${HIGH_PRICE}"
wait_for_sql "STOP activated as MARKET" "SELECT order_type FROM orders WHERE id='${ORDER_ID}'" "MARKET"
wait_for_sql "trigger timestamp recorded" "SELECT (triggered_at IS NOT NULL)::text FROM orders WHERE id='${ORDER_ID}'" "true"
wait_for_sql "order.triggered outbox event written" "SELECT count(*)::text FROM order_outbox WHERE aggregate_id='${ORDER_ID}' AND event_type='order.triggered'" "1"

log "status: $(curl -fsS "${ORDER_URL}/internal/stop-trigger/status")"
log "metrics:"
curl -fsS "${ORDER_URL}/metrics" | grep -E 'tradeops_stop_trigger|tradeops_reference_price_age' || true

log "PASS: stop order trigger demo completed"
