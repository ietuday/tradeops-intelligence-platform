#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_PATH="${CHART_PATH:-${ROOT_DIR}/infrastructure/helm/tradeops-platform}"
RELEASE_NAME="${RELEASE_NAME:-tradeops}"

echo "Validating TradeOps Helm chart"
echo "Chart path: ${CHART_PATH}"

if [ ! -d "${CHART_PATH}" ]; then
  echo "ERROR: chart path does not exist: ${CHART_PATH}" >&2
  exit 1
fi

if [ ! -f "${CHART_PATH}/Chart.yaml" ]; then
  echo "ERROR: missing Chart.yaml in ${CHART_PATH}" >&2
  exit 1
fi

if ! command -v helm >/dev/null 2>&1; then
  cat <<EOF
Helm is not installed; skipping Helm lint/template validation.

Install Helm and rerun:
  helm lint infrastructure/helm/tradeops-platform
  helm template tradeops infrastructure/helm/tradeops-platform

EOF
  exit 0
fi

helm lint "${CHART_PATH}"
helm template "${RELEASE_NAME}" "${CHART_PATH}" >/dev/null

echo "Helm validation passed."

