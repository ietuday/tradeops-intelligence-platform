#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${BASE_URL:-http://localhost:8080}"
RESULTS_DIR="${RESULTS_DIR:-${ROOT_DIR}/load-results}"
VUS="${VUS:-5}"
DURATION="${DURATION:-30s}"

usage() {
  cat <<'USAGE'
Usage: scripts/run-load-tests.sh [--gateway] [--orders] [--surveillance] [--notifications] [--audit] [--all]

Environment:
  BASE_URL=http://localhost:8080
  TOKEN=<jwt for protected APIs>
  VUS=5
  DURATION=30s
  RESULTS_DIR=load-results

Load tests are opt-in and local-safe by default. --all runs every scenario with low defaults.
USAGE
}

if ! command -v k6 >/dev/null 2>&1; then
  echo "k6 is not installed; skipping load tests."
  echo "Install k6 from https://grafana.com/docs/k6/latest/set-up/install-k6/ and rerun this script."
  exit 0
fi

if [ "$#" -eq 0 ]; then
  usage
  exit 0
fi

declare -a SELECTED=()

while [ "$#" -gt 0 ]; do
  case "$1" in
    --gateway)
      SELECTED+=("gateway-health.js")
      ;;
    --orders)
      SELECTED+=("order-flow.js")
      ;;
    --surveillance)
      SELECTED+=("surveillance-flow.js")
      ;;
    --notifications)
      SELECTED+=("notification-flow.js")
      ;;
    --audit)
      SELECTED+=("audit-flow.js")
      ;;
    --all)
      SELECTED=("gateway-health.js" "order-flow.js" "surveillance-flow.js" "notification-flow.js" "audit-flow.js")
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

mkdir -p "${RESULTS_DIR}"

echo "BASE_URL=${BASE_URL}"
echo "VUS=${VUS}"
echo "DURATION=${DURATION}"
if [ -z "${TOKEN:-}" ]; then
  echo "TOKEN is not set; protected requests will be skipped by the k6 scripts."
fi

for script in "${SELECTED[@]}"; do
  scenario="${script%.js}"
  summary="${RESULTS_DIR}/${scenario}-summary.json"
  echo
  echo "Running ${script}"
  BASE_URL="${BASE_URL}" TOKEN="${TOKEN:-}" VUS="${VUS}" DURATION="${DURATION}" k6 run \
    --summary-export "${summary}" \
    "${ROOT_DIR}/tests/load/k6/${script}"
  echo "Saved summary: ${summary}"
done
