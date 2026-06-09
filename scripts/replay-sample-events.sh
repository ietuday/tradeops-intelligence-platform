#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
REDPANDA_SERVICE="${REDPANDA_SERVICE:-redpanda}"
EXAMPLES_DIR="${ROOT_DIR}/docs/examples"
TENANT_ID="${TENANT_ID:-default-tenant}"
CORRELATION_ID="${CORRELATION_ID:-demo-correlation-123}"

usage() {
  cat <<EOF
Usage:
  ./scripts/replay-sample-events.sh --surveillance
  ./scripts/replay-sample-events.sh --notifications
  ./scripts/replay-sample-events.sh --audit
  ./scripts/replay-sample-events.sh --all
EOF
}

mode="${1:-}"
case "${mode}" in
  --surveillance|--notifications|--audit|--all) ;;
  *) usage; exit 1 ;;
esac

publish_event() {
  local topic="$1"
  local file="$2"

  echo "Publishing ${file} -> ${topic}"
  if [ ! -f "${file}" ]; then
    echo "ERROR: sample payload missing: ${file}" >&2
    return 1
  fi

  if command -v docker >/dev/null 2>&1 && docker compose -f "${COMPOSE_FILE}" ps "${REDPANDA_SERVICE}" >/dev/null 2>&1; then
    if node -e 'const fs = require("fs"); const data = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); data.correlationId = process.argv[2]; data.tenantId = process.argv[3]; process.stdout.write(JSON.stringify(data) + "\n");' "${file}" "${CORRELATION_ID}" "${TENANT_ID}" |
      docker compose -f "${COMPOSE_FILE}" exec -T "${REDPANDA_SERVICE}" rpk topic produce "${topic}" >/dev/null; then
      echo "OK: published ${topic}"
      return 0
    fi
  fi

  echo "WARN: automatic publish failed or Redpanda is unavailable."
  print_manual_command "${topic}" "${file}"
}

print_manual_command() {
  local topic="$1"
  local file="$2"
  local rel_file="${file#${ROOT_DIR}/}"

  cat <<EOF
Manual publish:
CORRELATION_ID=${CORRELATION_ID} TENANT_ID=${TENANT_ID} node -e 'const fs = require("fs"); const data = JSON.parse(fs.readFileSync("${rel_file}", "utf8")); data.correlationId = process.env.CORRELATION_ID; data.tenantId = process.env.TENANT_ID; process.stdout.write(JSON.stringify(data) + "\\n");' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce ${topic}
EOF
}

echo "Correlation ID: ${CORRELATION_ID}"
echo "Tenant ID: ${TENANT_ID}"

if [ "${mode}" = "--surveillance" ] || [ "${mode}" = "--all" ]; then
  publish_event "order.created" "${EXAMPLES_DIR}/surveillance/order-created-large-order.json"
fi

if [ "${mode}" = "--notifications" ] || [ "${mode}" = "--all" ]; then
  publish_event "surveillance.alert.created" "${EXAMPLES_DIR}/notifications/surveillance-alert-created-high.json"
fi

if [ "${mode}" = "--audit" ] || [ "${mode}" = "--all" ]; then
  publish_event "order.created" "${EXAMPLES_DIR}/audit/order-created-audit-event.json"
  publish_event "surveillance.alert.created" "${EXAMPLES_DIR}/audit/surveillance-alert-created-audit-event.json"
fi

echo "Replay request complete."
