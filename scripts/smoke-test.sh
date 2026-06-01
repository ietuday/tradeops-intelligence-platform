#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
MARKET_DATA_URL="${MARKET_DATA_URL:-http://localhost:8085}"
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
check_contains "Angular shell placeholder" "${SHELL_URL}" "TradeOps Intelligence Platform - Shell"
check_contains "React trading dashboard placeholder" "${DASHBOARD_URL}" "Trading Dashboard - Foundation Ready"

echo "Smoke tests passed."
