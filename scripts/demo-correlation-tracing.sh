#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
API_URL="${API_URL:-http://localhost:8080}"
EXAMPLE_FILE="${ROOT_DIR}/docs/examples/surveillance/order-created-large-order.json"
TENANT_ID="${TENANT_ID:-default-tenant}"
CORRELATION_ID="${CORRELATION_ID:-}"
PUBLISH_SAMPLE="false"

usage() {
  cat <<EOF
Usage:
  ./scripts/demo-correlation-tracing.sh
  ./scripts/demo-correlation-tracing.sh --publish-sample

Environment:
  CORRELATION_ID=<id>  Use a specific correlation ID.
EOF
}

if [ "${1:-}" = "--publish-sample" ]; then
  PUBLISH_SAMPLE="true"
elif [ "${1:-}" != "" ]; then
  usage
  exit 1
fi

if [ -z "${CORRELATION_ID}" ]; then
  if command -v uuidgen >/dev/null 2>&1; then
    CORRELATION_ID="$(uuidgen)"
  else
    CORRELATION_ID="demo-correlation-$(date +%s)"
  fi
fi

echo "TradeOps correlation tracing demo"
echo "Tenant ID: ${TENANT_ID}"
echo "Correlation ID: ${CORRELATION_ID}"
echo "API Gateway: ${API_URL}"
echo

echo "Checking API Gateway /health with X-Correlation-ID..."
curl -fsS -D - "${API_URL}/health" -H "X-Correlation-ID: ${CORRELATION_ID}" -o /tmp/tradeops-correlation-health.json | grep -i "x-correlation-id" || true
cat /tmp/tradeops-correlation-health.json
echo

publish_sample() {
  if command -v docker >/dev/null 2>&1 && docker compose -f "${COMPOSE_FILE}" ps redpanda >/dev/null 2>&1; then
    node -e 'const fs = require("fs"); const data = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); data.correlationId = process.argv[2]; data.tenantId = process.argv[3]; process.stdout.write(JSON.stringify(data) + "\n");' "${EXAMPLE_FILE}" "${CORRELATION_ID}" "${TENANT_ID}" |
      docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce order.created >/dev/null
    echo "Published sample order.created with correlationId=${CORRELATION_ID}"
  else
    print_manual_publish
  fi
}

print_manual_publish() {
  cat <<EOF
Manual sample publish:
CORRELATION_ID=${CORRELATION_ID} TENANT_ID=${TENANT_ID} node -e 'const fs = require("fs"); const data = JSON.parse(fs.readFileSync("docs/examples/surveillance/order-created-large-order.json", "utf8")); data.correlationId = process.env.CORRELATION_ID; data.tenantId = process.env.TENANT_ID; process.stdout.write(JSON.stringify(data) + "\\n");' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce order.created
EOF
}

if [ "${PUBLISH_SAMPLE}" = "true" ]; then
  publish_sample
else
  echo "Read-only mode: no sample event published."
  echo "Use --publish-sample to publish docs/examples/surveillance/order-created-large-order.json to order.created."
fi

cat <<EOF

Trace logs:
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway order-service surveillance-service notification-service audit-service | grep ${CORRELATION_ID}

Audit API query:
curl "http://localhost:8080/api/audit/logs?correlationId=${CORRELATION_ID}" \\
  -H "Authorization: Bearer \${TOKEN}"

DLQ inspection:
./scripts/replay-dlq-events.sh --topic surveillance.dlq --dry-run
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume surveillance.dlq -n 1

EOF
