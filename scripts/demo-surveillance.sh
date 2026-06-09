#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/infrastructure/docker/.env}"
API_URL="${API_URL:-http://localhost:8080}"
SURVEILLANCE_URL="${SURVEILLANCE_URL:-http://localhost:8090}"
EXAMPLES_DIR="${ROOT_DIR}/docs/examples/surveillance"
DEMO_USER_ID="${DEMO_USER_ID:-33333333-3333-4333-8333-333333333333}"
TENANT_ID="${TENANT_ID:-default-tenant}"
CORRELATION_ID="${CORRELATION_ID:-demo-correlation-123}"

if [ -f "${ENV_FILE}" ]; then
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a
fi

JWT_SECRET="${SURVEILLANCE_JWT_SECRET:-${IDENTITY_JWT_SECRET:-local_dev_jwt_secret_change_me_123456789}}"
TOKEN="${SURVEILLANCE_DEMO_TOKEN:-}"

echo "TradeOps surveillance demo"
echo "API Gateway: ${API_URL}"
echo "Surveillance service: ${SURVEILLANCE_URL}"
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
  email: "surveillance.demo@example.com",
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

print_manual_publish_commands() {
  cat <<EOF

Kafka publish was skipped or unavailable. To publish the LargeOrderRule demo event manually:

CORRELATION_ID=${CORRELATION_ID} TENANT_ID=${TENANT_ID} node -e 'const fs = require("fs"); const data = JSON.parse(fs.readFileSync("docs/examples/surveillance/order-created-large-order.json", "utf8")); data.correlationId = process.env.CORRELATION_ID; data.tenantId = process.env.TENANT_ID; process.stdout.write(JSON.stringify(data) + "\n");' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce order.created

For AbnormalPriceMovementRule, publish a baseline tick first and then the jump event:

printf '{"eventType":"market.tick.normalized","symbol":"AAPL","price":100,"volume":1000,"source":"demo","correlationId":"%s"}\n' "${CORRELATION_ID}" | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce market.ticks

node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync("docs/examples/surveillance/market-tick-price-jump.json", "utf8"))) + "\n");' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce market.ticks

EOF
}

find_open_large_order_alert() {
  curl -fsS "${API_URL}/api/surveillance/alerts?status=OPEN&alertType=LargeOrderRule&limit=10" \
    -H "Authorization: Bearer ${TOKEN}" |
    node -e '
const chunks = [];
process.stdin.on("data", (chunk) => chunks.push(chunk));
process.stdin.on("end", () => {
  const data = JSON.parse(Buffer.concat(chunks).toString("utf8"));
  const alert = (data.alerts || [])[0];
  if (!alert || !alert.id) process.exit(1);
  process.stdout.write(alert.id);
});
'
}

check_contains "surveillance-service /health" "${SURVEILLANCE_URL}/health" "surveillance-service"
check_contains "API Gateway /api/surveillance/health" "${API_URL}/api/surveillance/health" "surveillance-service"

if [ -z "${TOKEN}" ]; then
  TOKEN="$(make_demo_token)"
fi

echo "Listing current open surveillance alerts..."
curl -fsS "${API_URL}/api/surveillance/alerts?status=OPEN&limit=5" \
  -H "Authorization: Bearer ${TOKEN}"
echo

if [ "${SKIP_KAFKA_PUBLISH:-false}" = "true" ]; then
  print_manual_publish_commands
else
  if command -v docker >/dev/null 2>&1 && docker compose -f "${COMPOSE_FILE}" ps redpanda >/dev/null 2>&1; then
    echo "Publishing sample LargeOrderRule event to order.created..."
    publish_event "order.created" "${EXAMPLES_DIR}/order-created-large-order.json"
    echo "Published sample event."
  else
    echo "Docker Compose Redpanda is not available from this shell."
    print_manual_publish_commands
  fi
fi

ALERT_ID=""
for attempt in $(seq 1 20); do
  if ALERT_ID="$(find_open_large_order_alert 2>/dev/null)"; then
    break
  fi
  sleep 1
done

if [ -z "${ALERT_ID}" ]; then
  echo "No open LargeOrderRule alert was found yet."
  echo "Confirm Redpanda is running and publish the sample event manually if needed."
  print_manual_publish_commands
  exit 1
fi

echo "Found alert: ${ALERT_ID}"
echo "Acknowledging alert..."
curl -fsS -X POST "${API_URL}/api/surveillance/alerts/${ALERT_ID}/acknowledge" \
  -H "Authorization: Bearer ${TOKEN}" >/dev/null
echo "OK: OPEN -> ACKNOWLEDGED"

echo "Resolving alert..."
curl -fsS -X POST "${API_URL}/api/surveillance/alerts/${ALERT_ID}/resolve" \
  -H "Authorization: Bearer ${TOKEN}" >/dev/null
echo "OK: ACKNOWLEDGED -> RESOLVED"

echo "Alert summary:"
curl -fsS "${API_URL}/api/surveillance/alerts/summary" \
  -H "Authorization: Bearer ${TOKEN}"
echo

echo "Demo complete."
