#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_PATH="${CHART_PATH:-${ROOT_DIR}/deployments/helm/tradeops}"
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
  helm lint deployments/helm/tradeops
  helm template tradeops deployments/helm/tradeops -f deployments/helm/tradeops/values-local.yaml

EOF
  exit 0
fi

helm lint "${CHART_PATH}"
helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${CHART_PATH}/values-local.yaml" >/dev/null
helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${CHART_PATH}/values-staging.yaml" >/dev/null
helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${CHART_PATH}/values-production.yaml" >/dev/null

echo "Helm validation passed."
