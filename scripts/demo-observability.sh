#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
REDPANDA_CONSOLE_URL="${REDPANDA_CONSOLE_URL:-http://localhost:8081}"
DASHBOARD_DIR="${ROOT_DIR}/infrastructure/docker/grafana/dashboards"
ALERT_RULES_FILE="${ROOT_DIR}/infrastructure/docker/prometheus/rules/tradeops-alerts.yml"

echo "TradeOps observability demo"
echo "Prometheus: ${PROMETHEUS_URL}"
echo "Grafana: ${GRAFANA_URL}"
echo "Redpanda Console: ${REDPANDA_CONSOLE_URL}"
echo

check_url() {
  local name="$1"
  local url="$2"

  if command -v curl >/dev/null 2>&1 && curl -fsS "${url}" >/dev/null 2>&1; then
    echo "OK: ${name}"
  else
    echo "WARN: ${name} is not reachable from this shell: ${url}"
  fi
}

check_url "Prometheus readiness" "${PROMETHEUS_URL}/-/ready"
check_url "Prometheus targets API" "${PROMETHEUS_URL}/api/v1/targets"
check_url "Prometheus rules API" "${PROMETHEUS_URL}/api/v1/rules"

echo
echo "Provisioned dashboard files:"
find "${DASHBOARD_DIR}" -maxdepth 1 -name "tradeops-*.json" -print | sort

echo
echo "Alert rules:"
echo "${ALERT_RULES_FILE}"

echo
echo "Open these pages:"
echo "- Grafana dashboards: ${GRAFANA_URL}/dashboards"
echo "- Prometheus targets: ${PROMETHEUS_URL}/targets"
echo "- Prometheus rules: ${PROMETHEUS_URL}/rules"
echo "- Prometheus alerts: ${PROMETHEUS_URL}/alerts"
echo "- Redpanda topics: ${REDPANDA_CONSOLE_URL}/topics"

echo
echo "Safe Prometheus queries to paste:"
cat <<'EOF'
up
sum(rate(tradeops_api_gateway_http_requests_total[5m])) or vector(0)
sum(rate(tradeops_api_gateway_http_requests_total{status_code=~"5.."}[5m])) or vector(0)
histogram_quantile(0.95, sum by (le) (rate(tradeops_api_gateway_http_request_duration_seconds_bucket[5m]))) or vector(0)
sum(rate(surveillance_alerts_created_total[5m])) or vector(0)
sum(rate(notifications_created_total[5m])) or vector(0)
sum(rate(audit_logs_created_total[5m])) or vector(0)
sum(rate(notification_delivery_failures_total[5m])) or vector(0)
sum(rate(audit_events_deadlettered_total[5m])) or vector(0)
EOF

echo
echo "To generate non-mutating observability traffic, run smoke health checks:"
echo "  make smoke"
echo
echo "To generate demo business metrics, run the focused demos:"
echo "  ./scripts/demo-surveillance.sh"
echo "  ./scripts/demo-notifications.sh"
echo "  ./scripts/demo-audit.sh"
echo
echo "Demo complete. This script does not require JWTs and does not mutate platform state."
