#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-tradeops-local}"
KIND_CONFIG="${KIND_CONFIG:-${ROOT_DIR}/deployments/kind/cluster.yaml}"

if ! command -v kind >/dev/null 2>&1; then
  echo "ERROR: kind is required. Install Kind and rerun." >&2
  exit 1
fi
if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl is required." >&2
  exit 1
fi

if kind get clusters | grep -qx "${CLUSTER_NAME}"; then
  echo "Kind cluster ${CLUSTER_NAME} already exists."
else
  kind create cluster --name "${CLUSTER_NAME}" --config "${KIND_CONFIG}"
fi

kubectl cluster-info --context "kind-${CLUSTER_NAME}"
echo "Kind cluster ready: ${CLUSTER_NAME}"

