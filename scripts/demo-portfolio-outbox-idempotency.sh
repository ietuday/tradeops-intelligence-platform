#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
PORTFOLIO_URL="${PORTFOLIO_URL:-http://localhost:8087}"
DATABASE_URL="${PORTFOLIO_DATABASE_URL:-postgres://tradeops:tradeops@localhost:5432/tradeops?sslmode=disable}"
TENANT_ID="${TENANT_ID:-default-tenant}"
BUYER_ID="${BUYER_ID:-demo-buyer}"
SELLER_ID="${SELLER_ID:-demo-seller}"
SYMBOL="${SYMBOL:-AAPL}"
EXECUTION_ID="${EXECUTION_ID:-exec_demo_portfolio_outbox}"
EVENT_ID="${EVENT_ID:-evt_demo_portfolio_outbox}"
CORRELATION_ID="${CORRELATION_ID:-demo-portfolio-outbox}"
TOPIC="${PORTFOLIO_TRADE_TOPIC:-trade.executed}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-45}"

log() { printf '[portfolio-outbox-demo] %s\n' "$*"; }
fail() { log "FAIL: $*"; exit 1; }
require() { command -v "$1" >/dev/null 2>&1 || fail "$1 is required"; }

require curl
require psql
require docker

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

publish_trade() {
  local event_time
  event_time="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  printf '{"eventId":"%s","eventType":"trade.executed","eventVersion":"1.0","tenantId":"%s","correlationId":"%s","occurredAt":"%s","executionId":"%s","buyOrderId":"buy-demo","sellOrderId":"sell-demo","buyerUserId":"%s","sellerUserId":"%s","symbol":"%s","executionQuantity":10,"executionPrice":100,"currency":"USD","source":"order-service"}\n' \
    "${EVENT_ID}" "${TENANT_ID}" "${CORRELATION_ID}" "${event_time}" "${EXECUTION_ID}" "${BUYER_ID}" "${SELLER_ID}" "${SYMBOL}" |
    docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${TOPIC}" >/dev/null
}

log "checking portfolio service at ${PORTFOLIO_URL}"
curl -fsS "${PORTFOLIO_URL}/health" >/dev/null || fail "portfolio service health check failed"

log "seeding seller inventory for ${SYMBOL}"
psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 >/dev/null <<SQL
WITH seller AS (
  INSERT INTO portfolios (tenant_id, user_id)
  VALUES ('${TENANT_ID}', '${SELLER_ID}')
  ON CONFLICT (tenant_id, user_id) DO UPDATE SET updated_at = portfolios.updated_at
  RETURNING id
)
INSERT INTO cash_balances (portfolio_id, cash_balance)
SELECT id, 100000 FROM seller
ON CONFLICT (portfolio_id) DO NOTHING;

INSERT INTO portfolio_holdings (tenant_id, portfolio_id, user_id, symbol, quantity, average_buy_price)
SELECT '${TENANT_ID}', id, '${SELLER_ID}', '${SYMBOL}', 100, 90 FROM portfolios
WHERE tenant_id = '${TENANT_ID}' AND user_id = '${SELLER_ID}'
ON CONFLICT (portfolio_id, symbol) DO UPDATE
SET quantity = 100, average_buy_price = 90, updated_at = now();
SQL

log "publishing trade event ${EXECUTION_ID}"
publish_trade
wait_for_sql "buyer received 10 shares" "SELECT COALESCE(SUM(quantity),0)::int::text FROM portfolio_holdings WHERE tenant_id='${TENANT_ID}' AND user_id='${BUYER_ID}' AND symbol='${SYMBOL}'" "10"
wait_for_sql "portfolio.updated outbox row inserted" "SELECT count(*)::text FROM portfolio_outbox_events WHERE event_type='portfolio.updated' AND payload->>'sourceExecutionId'='${EXECUTION_ID}'" "1"

log "publishing duplicate trade event ${EXECUTION_ID}"
publish_trade
sleep 3
quantity="$(psql "${DATABASE_URL}" -Atc "SELECT COALESCE(SUM(quantity),0)::int::text FROM portfolio_holdings WHERE tenant_id='${TENANT_ID}' AND user_id='${BUYER_ID}' AND symbol='${SYMBOL}'")"
[[ "${quantity}" == "10" ]] || fail "duplicate changed buyer quantity to ${quantity}"
log "OK: duplicate did not update position twice"

log "outbox status: $(curl -fsS "${PORTFOLIO_URL}/internal/portfolio/outbox/status")"
log "consumer status: $(curl -fsS "${PORTFOLIO_URL}/internal/portfolio/consumer/status")"
log "metrics:"
curl -fsS "${PORTFOLIO_URL}/metrics" | grep -E 'tradeops_portfolio_(events|outbox)' || true

log "PASS: portfolio outbox idempotency demo completed"
