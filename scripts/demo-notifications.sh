#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/infrastructure/docker/.env}"
API_URL="${API_URL:-http://localhost:8080}"
NOTIFICATION_URL="${NOTIFICATION_URL:-http://localhost:8091}"
EXAMPLES_DIR="${ROOT_DIR}/docs/examples/notifications"
DEMO_USER_ID="${DEMO_USER_ID:-33333333-3333-4333-8333-333333333333}"

if [ -f "${ENV_FILE}" ]; then
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a
fi

JWT_SECRET="${NOTIFICATION_JWT_SECRET:-${IDENTITY_JWT_SECRET:-local_dev_jwt_secret_change_me_123456789}}"
TOKEN="${NOTIFICATION_DEMO_TOKEN:-}"

echo "TradeOps notification demo"
echo "API Gateway: ${API_URL}"
echo "Notification service: ${NOTIFICATION_URL}"

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
  email: "notification.demo@example.com",
  roles: ["trader"],
  iat: now,
  exp: now + 3600
};
const signingInput = `${b64url(header)}.${b64url(payload)}`;
const signature = crypto.createHmac("sha256", secret).update(signingInput).digest("base64url");
process.stdout.write(`${signingInput}.${signature}`);
' "${JWT_SECRET}" "${DEMO_USER_ID}"
}

publish_event() {
  local topic="$1"
  local file="$2"

  node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync(process.argv[1], "utf8"))) + "\n");' "${file}" |
    docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${topic}" >/dev/null
}

print_manual_publish_command() {
  cat <<EOF

Kafka publish was skipped or unavailable. To publish the demo notification event manually:

node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync("docs/examples/notifications/surveillance-alert-created-high.json", "utf8"))) + "\\n");' | \\
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce surveillance.alert.created

EOF
}

find_notification() {
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

check_contains "notification-service /health" "${NOTIFICATION_URL}/health" "notification-service"
check_contains "API Gateway /api/notifications/health" "${API_URL}/api/notifications/health" "notification-service"

if [ -z "${TOKEN}" ]; then
  TOKEN="$(make_demo_token)"
fi

echo "Listing current notifications..."
curl -fsS "${API_URL}/api/notifications?limit=5" \
  -H "Authorization: Bearer ${TOKEN}"
echo

if [ "${SKIP_KAFKA_PUBLISH:-false}" = "true" ]; then
  print_manual_publish_command
else
  if command -v docker >/dev/null 2>&1 && docker compose -f "${COMPOSE_FILE}" ps redpanda >/dev/null 2>&1; then
    echo "Publishing sample surveillance.alert.created event..."
    publish_event "surveillance.alert.created" "${EXAMPLES_DIR}/surveillance-alert-created-high.json"
    echo "Published sample event."
  else
    echo "Docker Compose Redpanda is not available from this shell."
    print_manual_publish_command
  fi
fi

NOTIFICATION_ID=""
for attempt in $(seq 1 20); do
  if NOTIFICATION_ID="$(find_notification 2>/dev/null)"; then
    break
  fi
  sleep 1
done

if [ -z "${NOTIFICATION_ID}" ]; then
  echo "No notification was found yet."
  echo "Confirm notification-service is running and publish the sample event manually if needed."
  print_manual_publish_command
  exit 1
fi

echo "Found notification: ${NOTIFICATION_ID}"
echo "Marking notification as read..."
curl -fsS -X POST "${API_URL}/api/notifications/${NOTIFICATION_ID}/mark-read" \
  -H "Authorization: Bearer ${TOKEN}" >/dev/null
echo "OK: notification marked READ"

echo "Notification summary:"
curl -fsS "${API_URL}/api/notifications/summary" \
  -H "Authorization: Bearer ${TOKEN}"
echo

echo "Demo complete."
