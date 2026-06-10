#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
TOKEN=${TOKEN:-}
TENANT_ID=${TENANT_ID:-default-tenant}
RULE_NAME=${RULE_NAME:-LargeOrderRule}
DEMO_THRESHOLD=${DEMO_THRESHOLD:-250000}
APPLY=false

usage() {
  cat <<USAGE
TradeOps surveillance rule configuration demo

Usage:
  TOKEN=<jwt> ./scripts/demo-rule-config.sh
  TOKEN=<jwt> ./scripts/demo-rule-config.sh --apply

Environment:
  BASE_URL        API Gateway base URL. Default: http://localhost:8080
  TOKEN           JWT with trading_admin or risk_manager role for --apply.
  TENANT_ID       Tenant header for local demos. Default: default-tenant
  RULE_NAME       Rule to demo. Default: LargeOrderRule
  DEMO_THRESHOLD  Temporary threshold for --apply. Default: 250000
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --apply)
      APPLY=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

if [ -z "${TOKEN}" ]; then
  echo "TOKEN is required for protected surveillance APIs."
  echo "Example: TOKEN=<jwt> ./scripts/demo-rule-config.sh"
  exit 1
fi

api() {
  local method=$1
  local path=$2
  local body=${3:-}
  if [ -n "${body}" ]; then
    curl -sS -X "${method}" "${BASE_URL}${path}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "X-Tenant-ID: ${TENANT_ID}" \
      -H "Content-Type: application/json" \
      -d "${body}"
  else
    curl -sS -X "${method}" "${BASE_URL}${path}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "X-Tenant-ID: ${TENANT_ID}"
  fi
}

extract_threshold() {
  node -e 'let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => { const data = JSON.parse(body); process.stdout.write(String(data.thresholdNumeric ?? "")); });'
}

extract_field() {
  local field=$1
  node -e 'const field = process.argv[1]; let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => { const data = JSON.parse(body); process.stdout.write(String(data[field] ?? "")); });' "${field}"
}

echo "Checking surveillance rule config API through API Gateway..."
api GET "/api/surveillance/rules" | node -e 'let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => { const data = JSON.parse(body); console.log(JSON.stringify(data, null, 2)); });'

echo
echo "Current ${RULE_NAME}:"
current_rule=$(api GET "/api/surveillance/rules/${RULE_NAME}")
printf '%s\n' "${current_rule}" | node -e 'let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => console.log(JSON.stringify(JSON.parse(body), null, 2)));'

if [ "${APPLY}" != "true" ]; then
  cat <<EOF

Dry run only. To mutate safely and restore the original threshold:
TOKEN=<jwt> ./scripts/demo-rule-config.sh --apply

Manual examples:
curl -X PUT "${BASE_URL}/api/surveillance/rules/${RULE_NAME}" \\
  -H "Authorization: Bearer <jwt>" \\
  -H "X-Tenant-ID: ${TENANT_ID}" \\
  -H "Content-Type: application/json" \\
  -d '{"enabled":true,"severity":"CRITICAL","thresholdNumeric":${DEMO_THRESHOLD}}'

curl -X POST "${BASE_URL}/api/surveillance/rules/${RULE_NAME}/disable" \\
  -H "Authorization: Bearer <jwt>" \\
  -H "X-Tenant-ID: ${TENANT_ID}"
EOF
  exit 0
fi

original_threshold=$(printf '%s\n' "${current_rule}" | extract_threshold)
original_enabled=$(printf '%s\n' "${current_rule}" | extract_field enabled)
original_severity=$(printf '%s\n' "${current_rule}" | extract_field severity)
if [ -z "${original_threshold}" ]; then
  echo "Could not read original threshold; refusing to mutate."
  exit 1
fi

echo
echo "Applying temporary threshold ${DEMO_THRESHOLD} to ${RULE_NAME}..."
api PUT "/api/surveillance/rules/${RULE_NAME}" "{\"enabled\":true,\"severity\":\"CRITICAL\",\"thresholdNumeric\":${DEMO_THRESHOLD}}" | node -e 'let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => console.log(JSON.stringify(JSON.parse(body), null, 2)));'

echo
echo "Disabling ${RULE_NAME}..."
api POST "/api/surveillance/rules/${RULE_NAME}/disable" | node -e 'let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => console.log(JSON.stringify(JSON.parse(body), null, 2)));'

echo
echo "Re-enabling ${RULE_NAME}..."
api POST "/api/surveillance/rules/${RULE_NAME}/enable" | node -e 'let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => console.log(JSON.stringify(JSON.parse(body), null, 2)));'

echo
echo "Restoring original enabled=${original_enabled}, severity=${original_severity}, threshold=${original_threshold}..."
api PUT "/api/surveillance/rules/${RULE_NAME}" "{\"enabled\":${original_enabled},\"severity\":\"${original_severity}\",\"thresholdNumeric\":${original_threshold}}" | node -e 'let body=""; process.stdin.on("data", c => body += c); process.stdin.on("end", () => console.log(JSON.stringify(JSON.parse(body), null, 2)));'

echo
echo "Rule config demo complete."
