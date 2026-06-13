#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_PATH="${CHART_PATH:-${ROOT_DIR}/deployments/helm/tradeops}"
RELEASE_NAME="${RELEASE_NAME:-tradeops}"

echo "Validating Kubernetes scripts"
bash -n "${ROOT_DIR}/scripts/k8s-cluster-create.sh"
bash -n "${ROOT_DIR}/scripts/k8s-deploy.sh"
bash -n "${ROOT_DIR}/scripts/k8s-smoke-test.sh"
bash -n "${ROOT_DIR}/scripts/k8s-status.sh"
bash -n "${ROOT_DIR}/scripts/k8s-destroy-local.sh"
bash -n "${ROOT_DIR}/scripts/demo-k8s-deployment.sh"

if ! command -v helm >/dev/null 2>&1; then
  echo "Helm is not installed; skipping helm lint/template validation."
  exit 0
fi

helm lint "${CHART_PATH}"
helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${CHART_PATH}/values-local.yaml" >/dev/null
helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${CHART_PATH}/values-staging.yaml" >/dev/null
helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${CHART_PATH}/values-production.yaml" >/dev/null
helm package "${CHART_PATH}" --destination /tmp >/dev/null

echo "Kubernetes validation passed."

