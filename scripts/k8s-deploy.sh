#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_PATH="${CHART_PATH:-${ROOT_DIR}/deployments/helm/tradeops}"
RELEASE_NAME="${RELEASE_NAME:-tradeops}"
NAMESPACE="${NAMESPACE:-tradeops}"
VALUES_FILE="${VALUES_FILE:-${CHART_PATH}/values-local.yaml}"
TIMEOUT="${HELM_TIMEOUT:-10m}"

if ! command -v helm >/dev/null 2>&1; then
  echo "ERROR: helm is required." >&2
  exit 1
fi
if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl is required." >&2
  exit 1
fi

kubectl cluster-info >/dev/null
helm dependency update "${CHART_PATH}" >/dev/null 2>&1 || true
helm upgrade --install "${RELEASE_NAME}" "${CHART_PATH}" \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  -f "${VALUES_FILE}" \
  --wait \
  --timeout "${TIMEOUT}"

kubectl -n "${NAMESPACE}" get deploy,svc,job,pod

