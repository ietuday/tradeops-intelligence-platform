#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/infrastructure/docker/.env}"
API_URL="${API_URL:-http://localhost:8080}"
AUDIT_URL="${AUDIT_URL:-http://localhost:8092}"
EXAMPLES_DIR="${ROOT_DIR}/docs/examples/audit"
DEMO_USER_ID="${DEMO_USER_ID:-33333333-3333-4333-8333-333333333333}"
TENANT_ID="${TENANT_ID:-default-tenant}"
CORRELATION_ID="${CORRELATION_ID:-demo-correlation-123}"

if [ -f "${ENV_FILE}" ]; then
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a
fi

JWT_SECRET="${AUDIT_JWT_SECRET:-${IDENTITY_JWT_SECRET:-}}"
TOKEN="${AUDIT_DEMO_TOKEN:-}"

echo "TradeOps audit demo"
echo "API Gateway: ${API_URL}"
echo "Audit service: ${AUDIT_URL}"
echo "Tenant ID: ${TENANT_ID}"
echo "Correlation ID: ${CORRELATION_ID}"

check_contains() {
  local name="$1"
  local url="$2"
  local expected="$3"

  echo "Checking ${name}..."
  curl -fsS "${url}" | grep -q "${expected}"
  echo "OK: ${name}"
}

make_demo_token() {
  if ! command -v node >/dev/null 2>&1; then
    echo "Node.js is required to mint a local demo JWT. Set AUDIT_DEMO_TOKEN to skip token generation." >&2
    return 1
  fi

  node -e '
const crypto = require("crypto");
const secret = process.argv[1];
const userId = process.argv[2];
function b64url(value) {
  return Buffer.from(JSON.stringify(value)).toString("base64url");
}
const now = Math.floor(Date.now() / 1000);
const header = { alg: "HS256", typ: "JWT" };
const payload = {
  iss: "tradeops-identity-service",
  sub: userId,
  tenantId: process.argv[3],
  email: "audit.demo@example.com",
  roles: ["risk_manager"],
  iat: now,
  exp: now + 3600
};
const signingInput = `${b64url(header)}.${b64url(payload)}`;
const signature = crypto.createHmac("sha256", secret).update(signingInput).digest("base64url");
process.stdout.write(`${signingInput}.${signature}`);
' "${JWT_SECRET}" "${DEMO_USER_ID}" "${TENANT_ID}"
}

publish_event() {
  local topic="$1"
  local file="$2"

  node -e 'const fs = require("fs"); const data = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); data.correlationId = process.argv[2]; data.tenantId = process.argv[3]; process.stdout.write(JSON.stringify(data) + "\n");' "${file}" "${CORRELATION_ID}" "${TENANT_ID}" |
    docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${topic}" >/dev/null
}

print_manual_publish_command() {
  cat <<EOF

Kafka publish was skipped or unavailable. To publish the audit demo event manually:

CORRELATION_ID=${CORRELATION_ID} TENANT_ID=${TENANT_ID} node -e 'const fs = require("fs"); const data = JSON.parse(fs.readFileSync("docs/examples/audit/order-created-audit-event.json", "utf8")); data.correlationId = process.env.CORRELATION_ID; data.tenantId = process.env.TENANT_ID; process.stdout.write(JSON.stringify(data) + "\\n");' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce order.created

EOF
}

print_token_help() {
  cat <<EOF

A JWT is required for audit list, summary, and export APIs.

Set one of the following before rerunning this demo:

  AUDIT_DEMO_TOKEN=<access token from /api/auth/login>
  AUDIT_JWT_SECRET=<same local secret used by identity/api services>
  IDENTITY_JWT_SECRET=<same local secret used by identity/api services>

When ${ENV_FILE} exists, this script also loads JWT secrets from that file.

EOF
}

check_contains "audit-service /health" "${AUDIT_URL}/health" "audit-service"
check_contains "API Gateway /api/audit/health" "${API_URL}/api/audit/health" "audit-service"
check_contains "API Gateway /api/audit/ready" "${API_URL}/api/audit/ready" "ready"

if [ -z "${TOKEN}" ]; then
  if [ -z "${JWT_SECRET}" ]; then
    print_token_help
    exit 1
  fi
  TOKEN="$(make_demo_token)"
fi

if [ "${SKIP_KAFKA_PUBLISH:-false}" = "true" ]; then
  print_manual_publish_command
else
  if command -v docker >/dev/null 2>&1 && docker compose -f "${COMPOSE_FILE}" ps redpanda >/dev/null 2>&1; then
    echo "Publishing sample order.created audit source event..."
    publish_event "order.created" "${EXAMPLES_DIR}/order-created-audit-event.json"
    echo "Published sample event."
  else
    echo "Docker Compose Redpanda is not available from this shell."
    print_manual_publish_command
  fi
fi

echo "Listing audit logs..."
curl -fsS "${API_URL}/api/audit/logs?limit=5" \
  -H "Authorization: Bearer ${TOKEN}"
echo

echo "Audit summary:"
curl -fsS "${API_URL}/api/audit/summary" \
  -H "Authorization: Bearer ${TOKEN}"
echo

echo "Audit export JSON:"
curl -fsS "${API_URL}/api/audit/export?format=json&limit=5" \
  -H "Authorization: Bearer ${TOKEN}"
echo

echo "Demo complete."
