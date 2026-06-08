#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/infrastructure/docker/.env}"
API_URL="${API_URL:-http://localhost:8080}"
SURVEILLANCE_URL="${SURVEILLANCE_URL:-http://localhost:8090}"
NOTIFICATION_URL="${NOTIFICATION_URL:-http://localhost:8091}"
SURVEILLANCE_EXAMPLES_DIR="${ROOT_DIR}/docs/examples/surveillance"
DEMO_USER_ID="${DEMO_USER_ID:-33333333-3333-4333-8333-333333333333}"

if [ -f "${ENV_FILE}" ]; then
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a
fi

JWT_SECRET="${E2E_JWT_SECRET:-${IDENTITY_JWT_SECRET:-}}"
TOKEN="${E2E_DEMO_TOKEN:-}"

echo "TradeOps end-to-end demo"
echo "API Gateway: ${API_URL}"
echo "Surveillance service: ${SURVEILLANCE_URL}"
echo "Notification service: ${NOTIFICATION_URL}"

step() {
  echo
  echo "==> $1"
}

check_contains() {
  local name="$1"
  local url="$2"
  local expected="$3"

  echo "Checking ${name}..."
  if curl -fsS "${url}" | grep -q "${expected}"; then
    echo "OK: ${name}"
    return 0
  fi

  echo "FAIL: ${name}"
  return 1
}

require_node() {
  if ! command -v node >/dev/null 2>&1; then
    echo "Node.js is required for local JWT creation and JSON parsing."
    echo "Set E2E_DEMO_TOKEN=<access token from /api/auth/login> to skip local token generation."
    return 1
  fi
}

make_demo_token() {
  require_node
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
  email: "e2e.demo@example.com",
  roles: ["trader", "risk_manager"],
  iat: now,
  exp: now + 3600
};
const signingInput = `${b64url(header)}.${b64url(payload)}`;
const signature = crypto.createHmac("sha256", secret).update(signingInput).digest("base64url");
process.stdout.write(`${signingInput}.${signature}`);
' "${JWT_SECRET}" "${DEMO_USER_ID}"
}

print_token_help() {
  cat <<EOF

A JWT is required for the end-to-end API steps.

Set one of the following before rerunning this demo:

  E2E_DEMO_TOKEN=<access token from /api/auth/login>
  E2E_JWT_SECRET=<same local secret used by identity/api services>
  IDENTITY_JWT_SECRET=<same local secret used by identity/api services>

When ${ENV_FILE} exists, this script also loads JWT secrets from that file.

EOF
}

publish_event() {
  local topic="$1"
  local file="$2"

  require_node
  node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync(process.argv[1], "utf8"))) + "\n");' "${file}" |
    docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${topic}" >/dev/null
}

print_manual_publish_command() {
  cat <<EOF

Kafka publish was skipped or unavailable. To publish the large-order event manually:

node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync("docs/examples/surveillance/order-created-large-order.json", "utf8"))) + "\\n");' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce order.created

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

find_sent_notification() {
  curl -fsS "${API_URL}/api/notifications?status=SENT&limit=10" \
    -H "Authorization: Bearer ${TOKEN}" |
    node -e '
const chunks = [];
process.stdin.on("data", (chunk) => chunks.push(chunk));
process.stdin.on("end", () => {
  const data = JSON.parse(Buffer.concat(chunks).toString("utf8"));
  const notification = (data.notifications || [])[0];
  if (!notification || !notification.id) process.exit(1);
  process.stdout.write(notification.id);
});
'
}

wait_for_id() {
  local name="$1"
  local command_name="$2"
  local id=""

  for _ in $(seq 1 25); do
    if id="$("${command_name}" 2>/dev/null)"; then
      echo "${id}"
      return 0
    fi
    sleep 1
  done

  echo "Timed out waiting for ${name}." >&2
  return 1
}

step "Health checks"
check_contains "API Gateway /health" "${API_URL}/health" "api-gateway"
check_contains "Identity through Gateway" "${API_URL}/api/auth/health" "identity-service"
check_contains "Market through Gateway" "${API_URL}/api/market/health" "market-data-service"
check_contains "Order through Gateway" "${API_URL}/api/orders/health" "order-service"
check_contains "Portfolio through Gateway" "${API_URL}/api/portfolio/health" "portfolio-service"
check_contains "Strategy through Gateway" "${API_URL}/api/strategies/health" "strategy-service"
check_contains "Risk through Gateway" "${API_URL}/api/risk/health" "risk-engine-service"
check_contains "Surveillance direct /health" "${SURVEILLANCE_URL}/health" "surveillance-service"
check_contains "Surveillance through Gateway" "${API_URL}/api/surveillance/health" "surveillance-service"
check_contains "Notification direct /health" "${NOTIFICATION_URL}/health" "notification-service"
check_contains "Notification through Gateway" "${API_URL}/api/notifications/health" "notification-service"
check_contains "Notification through Gateway /ready" "${API_URL}/api/notifications/ready" "ready"

step "Authentication"
if [ -z "${TOKEN}" ]; then
  if [ -z "${JWT_SECRET}" ]; then
    print_token_help
    exit 1
  fi
  TOKEN="$(make_demo_token)"
  echo "Minted local demo JWT for user ${DEMO_USER_ID}."
else
  echo "Using E2E_DEMO_TOKEN from the environment."
fi

step "Current surveillance alerts"
curl -fsS "${API_URL}/api/surveillance/alerts?limit=5" \
  -H "Authorization: Bearer ${TOKEN}"
echo

step "Publish large-order event"
if [ "${SKIP_KAFKA_PUBLISH:-false}" = "true" ]; then
  print_manual_publish_command
else
  if command -v docker >/dev/null 2>&1 && docker compose -f "${COMPOSE_FILE}" ps redpanda >/dev/null 2>&1; then
    publish_event "order.created" "${SURVEILLANCE_EXAMPLES_DIR}/order-created-large-order.json"
    echo "Published sample order.created event."
  else
    echo "Docker Compose Redpanda is not available from this shell."
    print_manual_publish_command
  fi
fi

step "Wait for surveillance alert"
ALERT_ID="$(wait_for_id "LargeOrderRule alert" find_open_large_order_alert)" || {
  print_manual_publish_command
  exit 1
}
echo "Found alert: ${ALERT_ID}"

step "Acknowledge alert"
curl -fsS -X POST "${API_URL}/api/surveillance/alerts/${ALERT_ID}/acknowledge" \
  -H "Authorization: Bearer ${TOKEN}" >/dev/null
echo "OK: alert acknowledged"

step "List surveillance alerts"
curl -fsS "${API_URL}/api/surveillance/alerts?limit=5" \
  -H "Authorization: Bearer ${TOKEN}"
echo

step "Wait for notification"
NOTIFICATION_ID="$(wait_for_id "notification" find_sent_notification)" || {
  echo "No notification was found yet. Check notification-service logs and surveillance alert topics."
  exit 1
}
echo "Found notification: ${NOTIFICATION_ID}"

step "List notifications"
curl -fsS "${API_URL}/api/notifications?limit=5" \
  -H "Authorization: Bearer ${TOKEN}"
echo

step "Mark notification as read"
curl -fsS -X POST "${API_URL}/api/notifications/${NOTIFICATION_ID}/mark-read" \
  -H "Authorization: Bearer ${TOKEN}" >/dev/null
echo "OK: notification marked READ"

step "Notification summary"
curl -fsS "${API_URL}/api/notifications/summary" \
  -H "Authorization: Bearer ${TOKEN}"
echo

echo
echo "End-to-end demo complete."
