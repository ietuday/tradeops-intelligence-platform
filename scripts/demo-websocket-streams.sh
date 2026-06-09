#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-infrastructure/docker/docker-compose.yml}"
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infrastructure/docker/.env.example}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
WS_BASE_URL="${WS_BASE_URL:-ws://localhost:8080}"
TENANT_ID="${TENANT_ID:-default-tenant}"
CORRELATION_ID="${CORRELATION_ID:-demo-websocket-$(date +%s)}"
TOKEN="${TOKEN:-}"
STREAM="all"
PUBLISH_SAMPLE=false

usage() {
  cat <<'USAGE'
Usage: scripts/demo-websocket-streams.sh [--market|--orders|--alerts|--notifications|--audit|--all] [--publish-sample]

Environment:
  GATEWAY_URL=http://localhost:8080
  WS_BASE_URL=ws://localhost:8080
  COMPOSE_ENV_FILE=infrastructure/docker/.env.example
  TENANT_ID=default-tenant
  TOKEN=<jwt required when WS_REQUIRE_AUTH=true>
  CORRELATION_ID=demo-websocket-123

Examples:
  ./scripts/demo-websocket-streams.sh
  TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --orders
  TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --alerts --publish-sample
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --market) STREAM="market" ;;
    --orders) STREAM="orders" ;;
    --alerts) STREAM="alerts" ;;
    --notifications) STREAM="notifications" ;;
    --audit) STREAM="audit" ;;
    --all) STREAM="all" ;;
    --publish-sample) PUBLISH_SAMPLE=true ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

path_for_stream() {
  case "$1" in
    all) printf '/ws' ;;
    market) printf '/ws/market' ;;
    orders) printf '/ws/orders' ;;
    alerts) printf '/ws/alerts' ;;
    notifications) printf '/ws/notifications' ;;
    audit) printf '/ws/audit' ;;
  esac
}

topic_for_stream() {
  case "$1" in
    market) printf 'market.ticks' ;;
    orders) printf 'order.filled' ;;
    alerts|all) printf 'surveillance.alert.created' ;;
    notifications) printf 'notification.created' ;;
    audit) printf 'audit.log.created' ;;
  esac
}

payload_for_stream() {
  case "$1" in
    market) printf 'docs/examples/surveillance/market-tick-price-jump.json' ;;
    orders) printf 'docs/examples/audit/order-created-audit-event.json' ;;
    alerts|all) printf 'docs/examples/notifications/surveillance-alert-created-high.json' ;;
    notifications) printf 'docs/examples/websocket/notification-created-stream-message.json' ;;
    audit) printf 'docs/examples/websocket/audit-log-created-stream-message.json' ;;
  esac
}

echo "Checking API Gateway health at ${GATEWAY_URL}/health"
curl -fsS "${GATEWAY_URL}/health" >/dev/null

path="$(path_for_stream "${STREAM}")"
ws_url="${WS_BASE_URL}${path}?correlationId=${CORRELATION_ID}"
if [ -n "${TOKEN}" ]; then
  ws_url="${ws_url}&token=${TOKEN}"
else
  echo "TOKEN is not set. If WS_REQUIRE_AUTH=true, the gateway will reject the WebSocket connection."
fi

echo
echo "WebSocket endpoints:"
echo "  ${WS_BASE_URL}/ws"
echo "  ${WS_BASE_URL}/ws/market"
echo "  ${WS_BASE_URL}/ws/orders"
echo "  ${WS_BASE_URL}/ws/alerts"
echo "  ${WS_BASE_URL}/ws/notifications"
echo "  ${WS_BASE_URL}/ws/audit"
echo
echo "Selected stream: ${STREAM}"
echo "Tenant ID: ${TENANT_ID}"
echo "Correlation ID: ${CORRELATION_ID}"
echo "Connect URL: ${ws_url}"

if [ "${PUBLISH_SAMPLE}" = true ]; then
  topic="$(topic_for_stream "${STREAM}")"
  payload="$(payload_for_stream "${STREAM}")"
  echo
  echo "Publishing sample payload ${payload} to topic ${topic}"
  if command -v docker >/dev/null 2>&1 && [ -f "${payload}" ]; then
    docker compose --env-file "${COMPOSE_ENV_FILE}" -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${topic}" < "${payload}" || {
      echo "Sample publish failed. Ensure the Docker Compose stack is running and Redpanda is healthy."
    }
  else
    echo "Docker or sample payload not available. Manual command:"
    echo "docker compose --env-file ${COMPOSE_ENV_FILE} -f ${COMPOSE_FILE} exec -T redpanda rpk topic produce ${topic} < ${payload}"
  fi
fi

if command -v websocat >/dev/null 2>&1; then
  echo
  echo "Opening stream with websocat. Press Ctrl+C to exit."
  websocat "${ws_url}"
else
  echo
  echo "websocat is not installed; skipping live connection."
  echo "Install websocat or use another WebSocket client, then connect to:"
  echo "${ws_url}"
fi
