#!/usr/bin/env bash
set -euo pipefail

DASHBOARD_DIR="frontend/trading-dashboard-react"
DASHBOARD_URL="${DASHBOARD_URL:-http://localhost:5173}"
COMPOSE_DASHBOARD_URL="${COMPOSE_DASHBOARD_URL:-http://localhost:4300}"
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
WS_BASE_URL="${WS_BASE_URL:-ws://localhost:8080}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
BUILD=false

for arg in "$@"; do
  case "$arg" in
    --build)
      BUILD=true
      ;;
    -h|--help)
      echo "Usage: $0 [--build]"
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 1
      ;;
  esac
done

if [[ ! -f "${DASHBOARD_DIR}/package.json" ]]; then
  echo "React dashboard package not found at ${DASHBOARD_DIR}."
  exit 1
fi

echo "TradeOps real-time dashboard"
echo "Dashboard dev URL: ${DASHBOARD_URL}"
echo "Dashboard Compose URL: ${COMPOSE_DASHBOARD_URL}"
echo "API Gateway: ${API_BASE_URL}"
echo "WebSocket: ${WS_BASE_URL}/ws"
echo "Grafana: ${GRAFANA_URL}"
echo "Jaeger: ${JAEGER_URL}"
echo
echo "Local run:"
echo "  cd ${DASHBOARD_DIR}"
echo "  npm install"
echo "  npm run dev"
echo
echo "Paste a JWT into the dashboard Access panel for protected admin/risk/WebSocket views."

if [[ "$BUILD" == "true" ]]; then
  echo
  echo "Building dashboard..."
  (
    cd "${DASHBOARD_DIR}"
    npm install
    npm run build
  )
fi
