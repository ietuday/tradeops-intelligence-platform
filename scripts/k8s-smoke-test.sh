#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-tradeops}"
RELEASE_NAME="${RELEASE_NAME:-tradeops}"
API_PORT="${API_PORT:-18080}"
DASHBOARD_PORT="${DASHBOARD_PORT:-14300}"
ANGULAR_PORT="${ANGULAR_PORT:-14200}"

cleanup() {
  jobs -p | xargs -r kill >/dev/null 2>&1 || true
}
trap cleanup EXIT

kubectl -n "${NAMESPACE}" get namespace "${NAMESPACE}" >/dev/null
kubectl -n "${NAMESPACE}" wait --for=condition=available deploy --all --timeout=300s

failed_jobs="$(kubectl -n "${NAMESPACE}" get jobs -o jsonpath='{range .items[?(@.status.failed>0)]}{.metadata.name}{"\n"}{end}')"
if [ -n "${failed_jobs}" ]; then
  echo "ERROR: failed jobs detected:"
  echo "${failed_jobs}"
  exit 1
fi

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

