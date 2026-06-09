#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for perf smoke checks."
  exit 1
fi

check_endpoint() {
  local service="$1"
  local endpoint="$2"
  local result
  result="$(curl -sS -o /dev/null -w '%{http_code} %{time_total}' "${BASE_URL}${endpoint}" || printf '000 0')"
  printf '%-24s %-36s %-8s %-10s\n' "${service}" "${endpoint}" ${result}
}

echo "BASE_URL=${BASE_URL}"
printf '%-24s %-36s %-8s %-10s\n' "service" "endpoint" "status" "time_total"
printf '%-24s %-36s %-8s %-10s\n' "-------" "--------" "------" "----------"

check_endpoint "api-gateway" "/health"
check_endpoint "api-gateway" "/ready"
check_endpoint "identity-service" "/api/auth/health"
check_endpoint "market-data-service" "/api/market/health"
check_endpoint "order-service" "/api/orders/health"
check_endpoint "portfolio-service" "/api/portfolio/health"
check_endpoint "strategy-service" "/api/strategies/health"
check_endpoint "risk-engine-service" "/api/risk/health"
check_endpoint "surveillance-service" "/api/surveillance/health"
check_endpoint "notification-service" "/api/notifications/health"
check_endpoint "audit-service" "/api/audit/health"
