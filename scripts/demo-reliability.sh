#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
API_URL="${API_URL:-http://localhost:8080}"
PORTFOLIO_URL="${PORTFOLIO_URL:-http://localhost:8087}"
SURVEILLANCE_URL="${SURVEILLANCE_URL:-http://localhost:8090}"
NOTIFICATION_URL="${NOTIFICATION_URL:-http://localhost:8091}"

echo "TradeOps reliability demo"
echo "Set RUN_RELIABILITY_DEMO=true to publish an intentionally malformed notification event."

check_endpoint() {
  local name="$1"
  local url="$2"

  echo "Checking ${name}..."
  curl -fsS "${url}" >/dev/null
  echo "OK: ${name}"
}

print_manual_commands() {
  cat <<EOF

Manual DLQ inspection commands:

docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume portfolio.dlq -n 1
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume surveillance.dlq -n 1
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume notification.dlq -n 1

Manual malformed event publish command:

printf '{"bad":true}\\n' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce surveillance.alert.created

EOF
}

check_endpoint "API Gateway /health" "${API_URL}/health"
check_endpoint "portfolio-service /health" "${PORTFOLIO_URL}/health"
check_endpoint "surveillance-service /health" "${SURVEILLANCE_URL}/health"
check_endpoint "notification-service /health" "${NOTIFICATION_URL}/health"
check_endpoint "API Gateway metrics" "${API_URL}/metrics"
check_endpoint "portfolio-service metrics" "${PORTFOLIO_URL}/metrics"
check_endpoint "surveillance-service metrics" "${SURVEILLANCE_URL}/metrics"
check_endpoint "notification-service metrics" "${NOTIFICATION_URL}/metrics"

if [ "${RUN_RELIABILITY_DEMO:-false}" != "true" ]; then
  print_manual_commands
  echo "Reliability checks complete. No events were published."
  exit 0
fi

if ! command -v docker >/dev/null 2>&1 || ! docker compose -f "${COMPOSE_FILE}" ps redpanda >/dev/null 2>&1; then
  echo "Docker Compose Redpanda is not available from this shell."
  print_manual_commands
  exit 1
fi

echo "Publishing malformed surveillance.alert.created event to exercise notification retry/DLQ..."
printf '{"bad":true}\n' | docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce surveillance.alert.created >/dev/null
echo "Published malformed event. Inspect notification.dlq after retries:"
echo "docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume notification.dlq -n 1"
