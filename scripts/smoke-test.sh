#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
MARKET_DATA_URL="${MARKET_DATA_URL:-http://localhost:8085}"
ORDER_URL="${ORDER_URL:-http://localhost:8086}"
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

echo "Smoke tests passed."
