#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_PATH="${CHART_PATH:-${ROOT_DIR}/deployments/helm/tradeops}"
RELEASE_NAME="${HELM_RELEASE:-${RELEASE_NAME:-tradeops}}"
NAMESPACE="${K8S_NAMESPACE:-${NAMESPACE:-tradeops}}"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-tradeops-local}"
VALUES_FILE="${VALUES_FILE:-${CHART_PATH}/values-local.yaml}"
TIMEOUT="${HELM_TIMEOUT:-15m}"
LOAD_IMAGES="${K8S_LOAD_IMAGES:-true}"
RECOVER_PENDING="${K8S_RECOVER_PENDING:-true}"

diagnostics() {
  echo ""
  echo "Deployment diagnostics for namespace ${NAMESPACE}:"
  helm status "${RELEASE_NAME}" -n "${NAMESPACE}" || true
  kubectl -n "${NAMESPACE}" get deployments,replicasets,pods,svc,job,hpa,pdb,ingress || true
  echo ""
  echo "Recent warning events:"
  kubectl -n "${NAMESPACE}" get events --field-selector type=Warning --sort-by=.lastTimestamp | tail -30 || true
  echo ""
  echo "Failed job logs:"
  for job in $(kubectl -n "${NAMESPACE}" get jobs -o jsonpath='{range .items[?(@.status.failed>0)]}{.metadata.name}{"\n"}{end}' 2>/dev/null); do
    echo "== job/${job} =="
    kubectl -n "${NAMESPACE}" logs "job/${job}" --tail=120 || true
  done
  echo ""
  echo "Non-ready pod diagnostics:"
  for pod in $(kubectl -n "${NAMESPACE}" get pods -o jsonpath='{range .items[?(@.status.phase!="Succeeded")]}{.metadata.name}{" "}{.status.phase}{" "}{range .status.containerStatuses[*]}{.ready}{" "}{.state.waiting.reason}{" "}{end}{"\n"}{end}' 2>/dev/null | awk '$0 !~ / true / {print $1}'); do
    echo "== pod/${pod} =="
    kubectl -n "${NAMESPACE}" describe pod "${pod}" | sed -n '/Events:/,$p' || true
    kubectl -n "${NAMESPACE}" logs "${pod}" --all-containers --tail=80 || true
  done
}

if ! command -v helm >/dev/null 2>&1; then
  echo "ERROR: helm is required." >&2
  exit 1
fi
if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl is required." >&2
  exit 1
fi
if ! command -v kind >/dev/null 2>&1; then
  echo "ERROR: kind is required for local Kubernetes deployment." >&2
  exit 1
fi

kubectl cluster-info >/dev/null
if ! kind get clusters | grep -qx "${KIND_CLUSTER_NAME}"; then
  echo "ERROR: Kind cluster ${KIND_CLUSTER_NAME} does not exist. Run make k8s-create-local first." >&2
  exit 1
fi

if [ "${LOAD_IMAGES}" = "true" ]; then
  "${ROOT_DIR}/scripts/k8s-load-images.sh"
fi

helm dependency update "${CHART_PATH}" >/dev/null 2>&1 || true

helm lint "${CHART_PATH}"
helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${VALUES_FILE}" >/tmp/tradeops-helm-render.yaml

release_status="$(helm status "${RELEASE_NAME}" -n "${NAMESPACE}" 2>/dev/null | awk -F': ' '/^STATUS:/ {print $2; exit}' || true)"
case "${release_status}" in
  pending-install|pending-upgrade|pending-rollback)
    echo "ERROR: Helm release ${RELEASE_NAME} is ${release_status}." >&2
    echo "A previous Helm operation is still pending or was interrupted." >&2
    echo "Inspect with: helm status ${RELEASE_NAME} -n ${NAMESPACE}" >&2
    if [ "${RECOVER_PENDING}" = "true" ]; then
      echo "Recovering local release by uninstalling ${RELEASE_NAME}; namespace and cluster are preserved." >&2
      helm uninstall "${RELEASE_NAME}" -n "${NAMESPACE}" --wait || true
    else
      echo "For local recovery, run: helm uninstall ${RELEASE_NAME} -n ${NAMESPACE}" >&2
      diagnostics
      exit 1
    fi
    ;;
esac

if ! helm upgrade --install "${RELEASE_NAME}" "${CHART_PATH}" \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  -f "${VALUES_FILE}" \
  --wait \
  --timeout "${TIMEOUT}"; then
  echo "ERROR: Helm deployment failed or timed out." >&2
  echo "Recovery command: helm upgrade --install ${RELEASE_NAME} ${CHART_PATH} -n ${NAMESPACE} -f ${VALUES_FILE} --wait --timeout ${TIMEOUT}" >&2
  diagnostics
  exit 1
fi

kubectl -n "${NAMESPACE}" get deploy,rs,svc,job,pod
