#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
MARKET_DATA_URL="${MARKET_DATA_URL:-http://localhost:8085}"
ORDER_URL="${ORDER_URL:-http://localhost:8086}"
PORTFOLIO_URL="${PORTFOLIO_URL:-http://localhost:8087}"
STRATEGY_URL="${STRATEGY_URL:-http://localhost:8088}"
RISK_URL="${RISK_URL:-http://localhost:8089}"
SURVEILLANCE_URL="${SURVEILLANCE_URL:-http://localhost:8090}"
NOTIFICATION_URL="${NOTIFICATION_URL:-http://localhost:8091}"
SHELL_URL="${SHELL_URL:-http://localhost:4200}"
DASHBOARD_URL="${DASHBOARD_URL:-http://localhost:4300}"

echo "Running smoke tests against ${API_URL}"

check_contains() {
  local name="$1"
  local url="$2"
  local expected="$3"

  echo "Checking ${name}..."
  curl -fsS "${url}" | grep -q "${expected}"
  echo "OK: ${name}"
}

check_contains "API Gateway /health" "${API_URL}/health" "api-gateway"
check_contains "API Gateway /ready" "${API_URL}/ready" "api-gateway"
check_contains "API Gateway /metrics" "${API_URL}/metrics" "process_cpu"
check_contains "API Gateway /api/auth/health" "${API_URL}/api/auth/health" "identity-service"
check_contains "API Gateway /api/auth/ready" "${API_URL}/api/auth/ready" "identity-service"
check_contains "Market Data Service /health" "${MARKET_DATA_URL}/health" "market-data-service"
check_contains "API Gateway /api/market/health" "${API_URL}/api/market/health" "market-data-service"
check_contains "API Gateway /api/market/ready" "${API_URL}/api/market/ready" "market-data-service"
check_contains "Order Service /health" "${ORDER_URL}/health" "order-service"
check_contains "API Gateway /api/orders/health" "${API_URL}/api/orders/health" "order-service"
check_contains "API Gateway /api/orders/ready" "${API_URL}/api/orders/ready" "order-service"
check_contains "Portfolio Service /health" "${PORTFOLIO_URL}/health" "portfolio-service"
check_contains "API Gateway /api/portfolio/health" "${API_URL}/api/portfolio/health" "portfolio-service"
check_contains "API Gateway /api/portfolio/ready" "${API_URL}/api/portfolio/ready" "portfolio-service"
check_contains "Strategy Service /health" "${STRATEGY_URL}/health" "strategy-service"
check_contains "API Gateway /api/strategies/health" "${API_URL}/api/strategies/health" "strategy-service"
check_contains "API Gateway /api/strategies/ready" "${API_URL}/api/strategies/ready" "strategy-service"
check_contains "Risk Engine Service /health" "${RISK_URL}/health" "risk-engine-service"
check_contains "API Gateway /api/risk/health" "${API_URL}/api/risk/health" "risk-engine-service"
check_contains "API Gateway /api/risk/ready" "${API_URL}/api/risk/ready" "risk-engine-service"
check_contains "Surveillance Service /health" "${SURVEILLANCE_URL}/health" "surveillance-service"
check_contains "API Gateway /api/surveillance/health" "${API_URL}/api/surveillance/health" "surveillance-service"
check_contains "API Gateway /api/surveillance/ready" "${API_URL}/api/surveillance/ready" "ready"
check_contains "Notification Service /health" "${NOTIFICATION_URL}/health" "notification-service"
check_contains "API Gateway /api/notifications/health" "${API_URL}/api/notifications/health" "notification-service"
check_contains "API Gateway /api/notifications/ready" "${API_URL}/api/notifications/ready" "ready"
check_contains "Angular shell placeholder" "${SHELL_URL}" "TradeOps Intelligence Platform - Shell"
check_contains "React trading dashboard placeholder" "${DASHBOARD_URL}" "Trading Dashboard - Foundation Ready"

echo "Checking order workflow through API Gateway..."
curl -sS -X POST "${API_URL}/api/auth/register" \
  -H "Content-Type: application/json" \
  --data-raw '{"email":"smoke.trader@example.com","password":"Password@123","fullName":"Smoke Trader"}' >/dev/null || true

LOGIN_RESPONSE="$(curl -fsS -X POST "${API_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  --data-raw '{"email":"smoke.trader@example.com","password":"Password@123"}')"
ACCESS_TOKEN="$(node -e 'const data = JSON.parse(process.argv[1]); if (!data.accessToken) process.exit(1); process.stdout.write(data.accessToken);' "${LOGIN_RESPONSE}")"
IDEMPOTENCY_KEY="smoke-order-$(date +%s)"
ORDER_PAYLOAD='{"symbol":"AAPL","side":"BUY","orderType":"MARKET","quantity":10,"limitPrice":null,"stopPrice":null}'
ORDER_RESPONSE="$(curl -fsS -X POST "${API_URL}/api/orders" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: ${IDEMPOTENCY_KEY}" \
  --data-raw "${ORDER_PAYLOAD}")"
DUPLICATE_RESPONSE="$(curl -fsS -X POST "${API_URL}/api/orders" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: ${IDEMPOTENCY_KEY}" \
  --data-raw "${ORDER_PAYLOAD}")"
node -e 'const first = JSON.parse(process.argv[1]); const second = JSON.parse(process.argv[2]); if (!first.id || first.id !== second.id || first.status !== "filled") process.exit(1);' "${ORDER_RESPONSE}" "${DUPLICATE_RESPONSE}"
echo "Checking portfolio holdings update..."
for attempt in $(seq 1 20); do
  HOLDINGS_RESPONSE="$(curl -fsS "${API_URL}/api/portfolio/holdings" -H "Authorization: Bearer ${ACCESS_TOKEN}")"
  if node -e 'const data = JSON.parse(process.argv[1]); const found = (data.holdings || []).some((h) => h.symbol === "AAPL" && h.quantity >= 10); process.exit(found ? 0 : 1);' "${HOLDINGS_RESPONSE}"; then
    echo "OK: portfolio holdings"
    break
  fi
  if [ "${attempt}" = "20" ]; then
    echo "Portfolio holding for AAPL was not updated in time"
    exit 1
  fi
  sleep 1
done

INVALID_RESPONSE="$(curl -fsS -X POST "${API_URL}/api/orders" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: ${IDEMPOTENCY_KEY}-invalid" \
  --data-raw '{"symbol":"AAPL","side":"BUY","orderType":"MARKET","quantity":0}')"
node -e 'const data = JSON.parse(process.argv[1]); if (data.status !== "rejected" && !data.error) process.exit(1);' "${INVALID_RESPONSE}"

LIMIT_RESPONSE="$(curl -fsS -X POST "${API_URL}/api/orders" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: ${IDEMPOTENCY_KEY}-limit" \
  --data-raw '{"symbol":"MSFT","side":"BUY","orderType":"LIMIT","quantity":5,"limitPrice":250,"stopPrice":null}')"
LIMIT_ORDER_ID="$(node -e 'const data = JSON.parse(process.argv[1]); if (!data.id || data.status !== "accepted") process.exit(1); process.stdout.write(data.id);' "${LIMIT_RESPONSE}")"
CANCEL_RESPONSE="$(curl -fsS -X POST "${API_URL}/api/orders/${LIMIT_ORDER_ID}/cancel" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")"
node -e 'const data = JSON.parse(process.argv[1]); if (data.status !== "cancelled") process.exit(1);' "${CANCEL_RESPONSE}"
echo "OK: order workflow"

echo "Checking strategy workflow through API Gateway..."
STRATEGY_RESPONSE="$(curl -fsS -X POST "${API_URL}/api/strategies" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  --data-raw '{"name":"Smoke MA Cross","symbol":"AAPL","strategyType":"MOVING_AVERAGE_CROSSOVER","parameters":{"shortWindow":1,"longWindow":2}}')"
STRATEGY_ID="$(node -e 'const data = JSON.parse(process.argv[1]); if (!data.id) process.exit(1); process.stdout.write(data.id);' "${STRATEGY_RESPONSE}")"

for attempt in $(seq 1 60); do
  BACKTEST_PAYLOAD="$(node -e 'const now = new Date(); const start = new Date(now.getTime() - 60 * 60 * 1000); process.stdout.write(JSON.stringify({startTime: start.toISOString(), endTime: now.toISOString(), initialCapital: 100000}));')"
  set +e
  BACKTEST_RESPONSE="$(curl -sS -X POST "${API_URL}/api/strategies/${STRATEGY_ID}/backtest" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    -H "x-correlation-id: smoke-strategy-${attempt}" \
    --data-raw "${BACKTEST_PAYLOAD}")"
  BACKTEST_EXIT=$?
  set -e
  if [ "${BACKTEST_EXIT}" -eq 0 ] && node -e 'const data = JSON.parse(process.argv[1]); process.exit(data.id && data.performance ? 0 : 1);' "${BACKTEST_RESPONSE}"; then
    break
  fi
  if [ "${attempt}" = "60" ]; then
    echo "Strategy backtest did not have enough market data in time"
    echo "${BACKTEST_RESPONSE}"
    exit 1
  fi
  sleep 1
done

PERFORMANCE_RESPONSE="$(curl -fsS "${API_URL}/api/strategies/${STRATEGY_ID}/performance" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")"
node -e 'const data = JSON.parse(process.argv[1]); if (typeof data.totalReturn !== "number" || typeof data.totalTrades !== "number") process.exit(1);' "${PERFORMANCE_RESPONSE}"
SIGNALS_RESPONSE="$(curl -fsS "${API_URL}/api/strategies/${STRATEGY_ID}/signals" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")"
node -e 'const data = JSON.parse(process.argv[1]); if (!Array.isArray(data) || data.length < 1) process.exit(1);' "${SIGNALS_RESPONSE}"
echo "OK: strategy workflow"

echo "Checking risk workflow through API Gateway..."
RISK_SCORE_RESPONSE="$(curl -fsS "${API_URL}/api/risk/portfolio/score" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")"
node -e 'const data = JSON.parse(process.argv[1]); if (typeof data.score !== "number" || !data.level || !data.factors) process.exit(1);' "${RISK_SCORE_RESPONSE}"
RISK_VAR_RESPONSE="$(curl -fsS "${API_URL}/api/risk/portfolio/var" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")"
node -e 'const data = JSON.parse(process.argv[1]); if (typeof data.valueAtRisk !== "number" || data.confidenceLevel !== 95) process.exit(1);' "${RISK_VAR_RESPONSE}"
RISK_RECOMMENDATIONS_RESPONSE="$(curl -fsS "${API_URL}/api/risk/recommendations" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")"
node -e 'const data = JSON.parse(process.argv[1]); if (!Array.isArray(data) || data.length < 1) process.exit(1);' "${RISK_RECOMMENDATIONS_RESPONSE}"
echo "OK: risk workflow"

echo "Smoke tests passed."
