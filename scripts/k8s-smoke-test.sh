#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-tradeops}"
RELEASE_NAME="${RELEASE_NAME:-tradeops}"
API_PORT="${API_PORT:-18080}"
DASHBOARD_PORT="${DASHBOARD_PORT:-14300}"
ANGULAR_PORT="${ANGULAR_PORT:-14200}"
APP_PREFIX="${RELEASE_NAME}-tradeops"

cleanup() {
  jobs -p | xargs -r kill >/dev/null 2>&1 || true
}
trap cleanup EXIT

fail_with_status() {
  echo "ERROR: $*" >&2
  echo ""
  kubectl -n "${NAMESPACE}" get deployments,replicasets,pods,services,jobs -o wide || true
  echo ""
  kubectl -n "${NAMESPACE}" get events --field-selector type=Warning --sort-by=.lastTimestamp | tail -20 || true
  exit 1
}

kubectl -n "${NAMESPACE}" get namespace "${NAMESPACE}" >/dev/null
if command -v helm >/dev/null 2>&1; then
  release_status="$(helm status "${RELEASE_NAME}" -n "${NAMESPACE}" 2>/dev/null | awk -F': ' '/^STATUS:/ {print $2; exit}' || true)"
  [ "${release_status}" = "deployed" ] || fail_with_status "Helm release ${RELEASE_NAME} is ${release_status:-missing}, expected deployed."
fi

failed_jobs="$(kubectl -n "${NAMESPACE}" get jobs -o jsonpath='{range .items[?(@.status.failed>0)]}{.metadata.name}{"\n"}{end}')"
if [ -n "${failed_jobs}" ]; then
  echo "ERROR: failed jobs detected:"
  echo "${failed_jobs}"
  exit 1
fi

kubectl -n "${NAMESPACE}" wait --for=condition=complete "job/${APP_PREFIX}-db-migrate" --timeout=300s \
  || fail_with_status "migration job did not complete."
if kubectl -n "${NAMESPACE}" get job "${APP_PREFIX}-db-seed" >/dev/null 2>&1; then
  kubectl -n "${NAMESPACE}" wait --for=condition=complete "job/${APP_PREFIX}-db-seed" --timeout=300s \
    || fail_with_status "seed job did not complete."
fi

bad_pods="$(kubectl -n "${NAMESPACE}" get pods -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.phase}{" "}{range .status.containerStatuses[*]}{.state.waiting.reason}{" "}{end}{"\n"}{end}' \
  | grep -E 'ImagePullBackOff|ErrImagePull|CreateContainerConfigError|CrashLoopBackOff| Error | Pending ' || true)"
if [ -n "${bad_pods}" ]; then
  echo "${bad_pods}"
  fail_with_status "one or more pods are in a blocked startup state."
fi

for deployment in \
  "${APP_PREFIX}-postgresql" \
  "${APP_PREFIX}-redis" \
  "${APP_PREFIX}-redpanda" \
  "${APP_PREFIX}-mosquitto"; do
  if kubectl -n "${NAMESPACE}" get deploy "${deployment}" >/dev/null 2>&1; then
    kubectl -n "${NAMESPACE}" wait --for=condition=available "deploy/${deployment}" --timeout=300s \
      || fail_with_status "${deployment} is not available."
  fi
done

kubectl -n "${NAMESPACE}" wait --for=condition=available deploy --all --timeout=300s \
  || fail_with_status "not all deployments are available."

kubectl -n "${NAMESPACE}" port-forward svc/api-gateway "${API_PORT}:8080" >/tmp/tradeops-api-pf.log 2>&1 &
kubectl -n "${NAMESPACE}" port-forward svc/trading-dashboard-react "${DASHBOARD_PORT}:8080" >/tmp/tradeops-dashboard-pf.log 2>&1 &
kubectl -n "${NAMESPACE}" port-forward svc/shell-angular "${ANGULAR_PORT}:8080" >/tmp/tradeops-angular-pf.log 2>&1 &
sleep 5

curl -fsS "http://127.0.0.1:${API_PORT}/health" >/dev/null
curl -fsS "http://127.0.0.1:${DASHBOARD_PORT}/" >/dev/null
curl -fsS "http://127.0.0.1:${ANGULAR_PORT}/" >/dev/null

unauth_status="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${API_PORT}/api/admin/health-summary")"
case "${unauth_status}" in
  401|403) ;;
  *) echo "ERROR: expected protected admin API to reject unauthenticated request, got ${unauth_status}" >&2; exit 1 ;;
esac

echo "Kubernetes smoke test passed."
