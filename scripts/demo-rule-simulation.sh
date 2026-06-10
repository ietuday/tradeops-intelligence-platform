#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_URL="${API_URL:-http://localhost:8080}"
SURVEILLANCE_URL="${SURVEILLANCE_URL:-http://localhost:8090}"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/infrastructure/docker/.env}"
TENANT_ID="${TENANT_ID:-default-tenant}"
DEMO_USER_ID="${DEMO_USER_ID:-33333333-3333-4333-8333-333333333333}"
RULE_NAME="${RULE_NAME:-LargeOrderRule}"
LOOKBACK_MINUTES="${LOOKBACK_MINUTES:-60}"

if [ -f "${ENV_FILE}" ]; then
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a
elif [ -f "${ROOT_DIR}/infrastructure/docker/.env.example" ]; then
  set -a
  # shellcheck disable=SC1091
  . "${ROOT_DIR}/infrastructure/docker/.env.example"
  set +a
fi

JWT_SECRET="${SURVEILLANCE_JWT_SECRET:-${IDENTITY_JWT_SECRET:-local_dev_jwt_secret_change_me_123456789}}"
TOKEN="${SURVEILLANCE_DEMO_TOKEN:-${TOKEN:-}}"

make_demo_token() {
  node -e '
const crypto = require("crypto");
const secret = process.argv[1];
const userId = process.argv[2];
const tenantId = process.argv[3];
function b64url(value) {
  return Buffer.from(JSON.stringify(value)).toString("base64url");
}
const now = Math.floor(Date.now() / 1000);
const header = { alg: "HS256", typ: "JWT" };
const payload = {
  iss: "tradeops-identity-service",
  sub: userId,
  tenantId,
  email: "rule.simulation.demo@example.com",
  roles: ["risk_manager"],
  iat: now,
  exp: now + 3600
};
const signingInput = `${b64url(header)}.${b64url(payload)}`;
const signature = crypto.createHmac("sha256", secret).update(signingInput).digest("base64url");
process.stdout.write(`${signingInput}.${signature}`);
' "${JWT_SECRET}" "${DEMO_USER_ID}" "${TENANT_ID}"
}

check_service() {
  local name="$1"
  local url="$2"
  echo "Checking ${name}..."
  curl -fsS "${url}" >/dev/null
  echo "OK: ${name}"
}

simulate() {
  local label="$1"
  local payload="$2"
  local output

  echo "Running simulation: ${label}"
  output="$(curl -fsS -X POST "${API_URL}/api/surveillance/rules/${RULE_NAME}/simulate" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    --data "${payload}")"
  echo "${output}"
  echo "${output}" | node -e '
const chunks = [];
process.stdin.on("data", chunk => chunks.push(chunk));
process.stdin.on("end", () => {
  const result = JSON.parse(Buffer.concat(chunks).toString("utf8"));
  process.stdout.write(String(result.wouldTriggerAlerts || 0));
});
'
}

extract_threshold() {
  node -e '
const chunks = [];
process.stdin.on("data", chunk => chunks.push(chunk));
process.stdin.on("end", () => {
  const rule = JSON.parse(Buffer.concat(chunks).toString("utf8"));
  process.stdout.write(String(rule.thresholdNumeric || rule.thresholdPercent || rule.thresholdCount || ""));
});
'
}

if ! command -v node >/dev/null 2>&1; then
  echo "node is required to create the local demo JWT and parse JSON output."
  exit 1
fi

echo "TradeOps surveillance rule simulation demo"
echo "API Gateway: ${API_URL}"
echo "Surveillance service: ${SURVEILLANCE_URL}"
echo "Tenant ID: ${TENANT_ID}"
echo "Rule: ${RULE_NAME}"

check_service "surveillance-service /health" "${SURVEILLANCE_URL}/health"
check_service "API Gateway /api/surveillance/health" "${API_URL}/api/surveillance/health"

if [ -z "${TOKEN}" ]; then
  TOKEN="$(make_demo_token)"
fi

echo "Fetching current live rule config..."
CURRENT_CONFIG="$(curl -fsS "${API_URL}/api/surveillance/rules/${RULE_NAME}" \
  -H "Authorization: Bearer ${TOKEN}")"
echo "${CURRENT_CONFIG}"
CURRENT_THRESHOLD="$(printf "%s" "${CURRENT_CONFIG}" | extract_threshold)"
echo "Current live threshold: ${CURRENT_THRESHOLD:-unknown}"

CURRENT_PAYLOAD="$(printf '{"tenantId":"%s","ruleName":"%s","lookbackMinutes":%s,"dryRun":true}' "${TENANT_ID}" "${RULE_NAME}" "${LOOKBACK_MINUTES}")"
STRICT_PAYLOAD="$(printf '{"tenantId":"%s","ruleName":"%s","config":{"thresholdNumeric":300000},"lookbackMinutes":%s,"dryRun":true}' "${TENANT_ID}" "${RULE_NAME}" "${LOOKBACK_MINUTES}")"
RELAXED_PAYLOAD="$(printf '{"tenantId":"%s","ruleName":"%s","config":{"thresholdNumeric":50000},"lookbackMinutes":%s,"dryRun":true}' "${TENANT_ID}" "${RULE_NAME}" "${LOOKBACK_MINUTES}")"

CURRENT_ALERTS="$(simulate "current config" "${CURRENT_PAYLOAD}" | tail -n 1)"
STRICT_ALERTS="$(simulate "stricter threshold" "${STRICT_PAYLOAD}" | tail -n 1)"
RELAXED_ALERTS="$(simulate "relaxed threshold" "${RELAXED_PAYLOAD}" | tail -n 1)"

cat <<EOF

Simulation comparison:
  current config:     ${CURRENT_ALERTS} would-trigger alert(s)
  stricter threshold: ${STRICT_ALERTS} would-trigger alert(s)
  relaxed threshold:  ${RELAXED_ALERTS} would-trigger alert(s)

Re-fetching live config to confirm the dry run did not mutate it...
EOF

AFTER_CONFIG="$(curl -fsS "${API_URL}/api/surveillance/rules/${RULE_NAME}" \
  -H "Authorization: Bearer ${TOKEN}")"
AFTER_THRESHOLD="$(printf "%s" "${AFTER_CONFIG}" | extract_threshold)"
echo "Live threshold after simulation: ${AFTER_THRESHOLD:-unknown}"

if [ "${CURRENT_THRESHOLD}" != "${AFTER_THRESHOLD}" ]; then
  echo "FAIL: live config threshold changed during simulation."
  exit 1
fi

echo "OK: live rule config remains unchanged."
