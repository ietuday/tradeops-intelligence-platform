#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"

if [[ -z "${TOKEN:-}" ]]; then
  echo "TOKEN is required for admin APIs."
  echo "Example: TOKEN=<jwt> API_BASE_URL=${API_BASE_URL} $0"
  exit 1
fi

tenant_query=""
if [[ -n "${TENANT_ID:-}" ]]; then
  tenant_query="?tenantId=${TENANT_ID}"
fi

curl_admin() {
  local path="$1"
  echo
  echo "GET ${API_BASE_URL}${path}"
  curl -sS \
    -H "authorization: Bearer ${TOKEN}" \
    -H "accept: application/json" \
    "${API_BASE_URL}${path}"
  echo
}

curl_admin "/api/admin/health-summary"
curl_admin "/api/admin/services"
curl_admin "/api/admin/topics"
curl_admin "/api/admin/dlq-summary"
curl_admin "/api/admin/alerts-summary${tenant_query}"
curl_admin "/api/admin/notifications-summary${tenant_query}"
curl_admin "/api/admin/audit-summary${tenant_query}"
curl_admin "/api/admin/rule-config-summary${tenant_query}"
curl_admin "/api/admin/platform-config"
curl_admin "/api/admin/ops-checklist"
