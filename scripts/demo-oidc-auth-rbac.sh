#!/usr/bin/env bash
set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
IDENTITY_URL="${IDENTITY_URL:-http://localhost:8081}"

echo "== OIDC discovery =="
curl -fsS "${IDENTITY_URL}/.well-known/openid-configuration"
echo

echo "== JWKS =="
curl -fsS "${IDENTITY_URL}/.well-known/jwks.json"
echo

login() {
  local email="$1"
  local password="${2:-Password123!}"
  curl -fsS -X POST "${GATEWAY_URL}/api/auth/login" \
    -H 'content-type: application/json' \
    --data "{\"email\":\"${email}\",\"password\":\"${password}\"}" |
    node -e "let b='';process.stdin.on('data',d=>b+=d);process.stdin.on('end',()=>console.log(JSON.parse(b).accessToken||''))"
}

TRADER_TOKEN="${TRADER_TOKEN:-$(login "${TRADER_EMAIL:-trader@tradeops.local}" "${TRADER_PASSWORD:-Password123!}")}"
ADMIN_TOKEN="${ADMIN_TOKEN:-$(login "${ADMIN_EMAIL:-admin@tradeops.local}" "${ADMIN_PASSWORD:-Password123!}")}"

echo "== trader can read orders =="
curl -i -sS "${GATEWAY_URL}/api/orders" -H "authorization: Bearer ${TRADER_TOKEN}" | sed -n '1,12p'

echo "== trader admin/audit request should be forbidden =="
curl -i -sS "${GATEWAY_URL}/api/audit/summary" -H "authorization: Bearer ${TRADER_TOKEN}" | sed -n '1,12p'

echo "== admin can read audit summary =="
curl -i -sS "${GATEWAY_URL}/api/audit/summary" -H "authorization: Bearer ${ADMIN_TOKEN}" | sed -n '1,12p'

echo "== spoofed identity header is stripped at gateway boundary =="
curl -i -sS "${GATEWAY_URL}/api/portfolio" \
  -H "authorization: Bearer ${TRADER_TOKEN}" \
  -H "x-tradeops-tenant-id: attacker-tenant" | sed -n '1,12p'

echo "== auth metrics =="
curl -fsS "${GATEWAY_URL}/metrics" | grep -E 'tradeops_auth_|tradeops_service_auth_' | head -40 || true

