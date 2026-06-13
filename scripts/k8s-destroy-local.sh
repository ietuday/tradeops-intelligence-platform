#!/usr/bin/env bash
set -Eeuo pipefail

CLUSTER_NAME="${KIND_CLUSTER_NAME:-tradeops-local}"
NAMESPACE="${NAMESPACE:-tradeops}"

if [ "${DELETE_KIND_CLUSTER:-false}" = "true" ]; then
  if ! command -v kind >/dev/null 2>&1; then
    echo "ERROR: kind is required to delete the local cluster." >&2
    exit 1
  fi
  kind delete cluster --name "${CLUSTER_NAME}"
  exit 0
fi

if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl is required." >&2
  exit 1
fi

kubectl delete namespace "${NAMESPACE}" --ignore-not-found
echo "Namespace ${NAMESPACE} deleted. Set DELETE_KIND_CLUSTER=true to delete Kind cluster ${CLUSTER_NAME}."

