#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
TENANT_ID="${TENANT_ID:-default-tenant}"
CORRELATION_ID="${CORRELATION_ID:-demo-correlation-123}"
RUN=false

for arg in "$@"; do
  case "$arg" in
    --run|--apply)
      RUN=true
      ;;
    -h|--help)
      echo "Usage: TOKEN=<jwt> $0 [--run]"
      echo "Prints read-only curl examples by default. --run sends sample analytics requests."
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 1
      ;;
  esac
done

headers=(-H "accept: application/json" -H "x-tenant-id: ${TENANT_ID}" -H "x-correlation-id: ${CORRELATION_ID}")
if [[ -n "${TOKEN:-}" ]]; then
  headers+=(-H "authorization: Bearer ${TOKEN}")
fi

print_curl() {
  local method="$1"
  local path="$2"
  local payload="${3:-}"
  echo
  echo "${method} ${API_BASE_URL}${path}"
  if [[ -n "$payload" ]]; then
    echo "curl -sS -X ${method} ${API_BASE_URL}${path} -H 'authorization: Bearer <token>' -H 'x-tenant-id: ${TENANT_ID}' -H 'x-correlation-id: ${CORRELATION_ID}' -H 'content-type: application/json' --data @${payload}"
  else
    echo "curl -sS ${API_BASE_URL}${path} -H 'authorization: Bearer <token>' -H 'x-tenant-id: ${TENANT_ID}' -H 'x-correlation-id: ${CORRELATION_ID}'"
  fi
}

run_curl() {
  local method="$1"
  local path="$2"
  local payload="${3:-}"
  if [[ -n "$payload" ]]; then
    curl -sS -X "$method" "${API_BASE_URL}${path}" "${headers[@]}" -H "content-type: application/json" --data @"$payload"
  else
    curl -sS "${API_BASE_URL}${path}" "${headers[@]}"
  fi
  echo
}

print_curl GET "/api/risk/scenarios"
print_curl POST "/api/risk/stress-test" "docs/examples/risk/stress-test-request.json"
print_curl POST "/api/risk/portfolio/concentration" "docs/examples/risk/concentration-request.json"
print_curl POST "/api/risk/portfolio/drawdown-trend" "docs/examples/risk/drawdown-request.json"
print_curl POST "/api/risk/volatility-shock"

if [[ "$RUN" != "true" ]]; then
  echo
  echo "Dry run only. Re-run with --run to send requests."
  exit 0
fi

run_curl GET "/api/risk/scenarios"
run_curl POST "/api/risk/stress-test" "docs/examples/risk/stress-test-request.json"
run_curl POST "/api/risk/portfolio/concentration" "docs/examples/risk/concentration-request.json"
run_curl POST "/api/risk/portfolio/drawdown-trend" "docs/examples/risk/drawdown-request.json"
run_curl POST "/api/risk/volatility-shock" "docs/examples/risk/volatility-shock-request.json"
