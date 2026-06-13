#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-tradeops}"
RELEASE_NAME="${RELEASE_NAME:-tradeops}"

kubectl -n "${NAMESPACE}" get namespace "${NAMESPACE}" >/dev/null
if command -v helm >/dev/null 2>&1; then
  helm status "${RELEASE_NAME}" -n "${NAMESPACE}" || true
fi
kubectl -n "${NAMESPACE}" get pods,deployments,services,ingress,jobs,hpa,pdb
echo ""
echo "Recent warning events:"
kubectl -n "${NAMESPACE}" get events --field-selector type=Warning --sort-by=.lastTimestamp | tail -20 || true
echo ""
echo "Non-ready pod logs:"
for pod in $(kubectl -n "${NAMESPACE}" get pods --field-selector=status.phase!=Running -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null); do
  echo "== ${pod} =="
  kubectl -n "${NAMESPACE}" logs "${pod}" --tail=80 || true
done

