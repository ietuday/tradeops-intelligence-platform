#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${BASE_URL:-http://localhost:8080}"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
CORRELATION_ID="${CORRELATION_ID:-otel-demo-$(date +%s)}"
TENANT_ID="${TENANT_ID:-default-tenant}"
CREATE_ORDER=false

for arg in "$@"; do
  case "$arg" in
    --create-order) CREATE_ORDER=true ;;
    -h|--help)
      cat <<HELP
Usage: CORRELATION_ID=<id> TOKEN=<jwt> $0 [--create-order]

Safe by default: prints tracing commands and checks Jaeger. Use --create-order
to submit a demo order through the API Gateway.
HELP
      exit 0
      ;;
    *) echo "Unknown argument: $arg" >&2; exit 1 ;;
  esac
done

echo "OpenTelemetry tracing demo"
echo "Jaeger UI: ${JAEGER_URL}"
echo "Correlation ID: ${CORRELATION_ID}"
echo "Tenant ID: ${TENANT_ID}"
echo

if curl -fsS "${JAEGER_URL}" >/dev/null 2>&1; then
  echo "Jaeger UI reachable"
else
  echo "Jaeger UI not reachable yet. Start the platform with:"
  echo "  docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml up -d"
fi

echo
echo "Trace context:"
echo "  OpenTelemetry uses W3C traceparent/tracestate headers."
echo "  You normally do not need to create traceparent manually; the gateway starts/extracts spans."
echo "  X-Correlation-ID remains the stable log/event/audit lookup key."

echo
echo "Health check with correlation ID:"
echo "  curl -i ${BASE_URL}/api/orders/health -H 'X-Correlation-ID: ${CORRELATION_ID}' -H 'X-Tenant-ID: ${TENANT_ID}'"
curl -fsS "${BASE_URL}/api/orders/health" \
  -H "X-Correlation-ID: ${CORRELATION_ID}" \
  -H "X-Tenant-ID: ${TENANT_ID}" >/dev/null 2>&1 || true

if [[ "${CREATE_ORDER}" == "true" ]]; then
  if [[ -z "${TOKEN:-}" ]]; then
    echo
    echo "TOKEN is required for --create-order."
    echo "Login first and export TOKEN, then rerun:"
    echo "  TOKEN=<jwt> CORRELATION_ID=${CORRELATION_ID} $0 --create-order"
  else
    echo
    echo "Creating demo order through API Gateway..."
    curl -fsS -X POST "${BASE_URL}/api/orders" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -H "Idempotency-Key: otel-${CORRELATION_ID}" \
      -H "X-Correlation-ID: ${CORRELATION_ID}" \
      -H "X-Tenant-ID: ${TENANT_ID}" \
      -d '{"symbol":"AAPL","side":"BUY","orderType":"MARKET","quantity":1}' || true
    echo
  fi
else
  echo
  echo "Mutation skipped. Add --create-order with TOKEN=<jwt> to trigger the order flow."
fi

echo
echo "Find the trace in Jaeger:"
echo "  1. Open ${JAEGER_URL}"
echo "  2. Select service: api-gateway"
echo "  3. Search operations like GET /api/orders/health or POST /api/orders"
echo
echo "Find related logs by correlation ID:"
echo "  docker compose --env-file infrastructure/docker/.env.example -f ${ROOT_DIR}/infrastructure/docker/docker-compose.yml logs api-gateway order-service surveillance-service notification-service audit-service | grep ${CORRELATION_ID}"
