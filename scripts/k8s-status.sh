#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-tradeops}"
RELEASE_NAME="${RELEASE_NAME:-tradeops}"

echo "Current Kubernetes context:"
kubectl config current-context
echo ""
echo "Cluster nodes:"
kubectl get nodes -o wide
echo ""
kubectl -n "${NAMESPACE}" get namespace "${NAMESPACE}" >/dev/null
if command -v helm >/dev/null 2>&1; then
  echo "Helm release:"
  helm status "${RELEASE_NAME}" -n "${NAMESPACE}" || true
  echo ""
fi
echo "Workloads:"
kubectl -n "${NAMESPACE}" get deployments,replicasets,pods,services,ingress,jobs,hpa,pdb -o wide
echo ""
echo "Recent warning events:"
kubectl -n "${NAMESPACE}" get events --field-selector type=Warning --sort-by=.lastTimestamp | tail -20 || true
echo ""
echo "Pods with waiting/error states:"
kubectl -n "${NAMESPACE}" get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\t"}{range .spec.containers[*]}{.image}{","}{end}{"\t"}{range .status.containerStatuses[*]}{.restartCount}{":"}{.state.waiting.reason}{":"}{.lastState.terminated.reason}{" "}{end}{"\n"}{end}' \
  | awk '$0 ~ /ImagePullBackOff|ErrImagePull|CreateContainerConfigError|CrashLoopBackOff|RunContainerError|Pending/ {print}' || true
echo ""
echo "Diagnostics for non-ready pods:"
for pod in $(kubectl -n "${NAMESPACE}" get pods -o jsonpath='{range .items[?(@.status.phase!="Succeeded")]}{.metadata.name}{" "}{range .status.containerStatuses[*]}{.ready}{" "}{.state.waiting.reason}{" "}{end}{"\n"}{end}' 2>/dev/null | awk '$0 !~ / true / {print $1}'); do
  echo "== ${pod} =="
  kubectl -n "${NAMESPACE}" describe pod "${pod}" | sed -n '/Events:/,$p' || true
  kubectl -n "${NAMESPACE}" logs "${pod}" --all-containers --tail=80 || true
done
