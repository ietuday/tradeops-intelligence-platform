#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-tradeops-local}"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/infrastructure/docker/.env.example}"
BUILD_IMAGES="${K8S_BUILD_IMAGES:-true}"
LOAD_IMAGES="${K8S_LOAD_IMAGES:-true}"

SERVICES=(
  api-gateway
  audit-service
  identity-service
  market-data-service
  notification-service
  order-service
  portfolio-service
  risk-engine-service
  shell-angular
  strategy-service
  surveillance-service
  trading-dashboard-react
)

IMAGES=(
  tradeops/api-gateway:0.1.0
  tradeops/audit-service:1.3.0
  tradeops/identity-service:0.2.0
  tradeops/market-data-service:0.3.0
  tradeops/notification-service:0.9.0
  tradeops/order-service:0.4.0
  tradeops/portfolio-service:0.5.0
  tradeops/risk-engine-service:0.7.0
  tradeops/shell-angular:0.1.0
  tradeops/strategy-service:0.6.0
  tradeops/surveillance-service:0.8.0
  tradeops/trading-dashboard-react:0.1.0
)

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: $1 is required." >&2
    exit 1
  fi
}

image_exists() {
  docker image inspect "$1" >/dev/null 2>&1
}

require_command docker

if [ "${LOAD_IMAGES}" = "true" ]; then
  require_command kind
  if ! kind get clusters | grep -qx "${CLUSTER_NAME}"; then
    echo "ERROR: Kind cluster ${CLUSTER_NAME} does not exist. Run make k8s-create-local first." >&2
    exit 1
  fi
fi

if [ "${BUILD_IMAGES}" = "true" ]; then
  echo "Building TradeOps application images with Docker Compose"
  docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" build "${SERVICES[@]}"
fi

missing=()
for image in "${IMAGES[@]}"; do
  if ! image_exists "${image}"; then
    missing+=("${image}")
  fi
done

if [ "${#missing[@]}" -gt 0 ]; then
  echo "ERROR: required local images are missing:" >&2
  printf '  %s\n' "${missing[@]}" >&2
  echo "Run with K8S_BUILD_IMAGES=true or build the missing images manually." >&2
  exit 1
fi

if [ "${LOAD_IMAGES}" = "true" ]; then
  echo "Loading TradeOps application images into Kind cluster ${CLUSTER_NAME}"
  kind load docker-image "${IMAGES[@]}" --name "${CLUSTER_NAME}"
else
  echo "K8S_LOAD_IMAGES=false; skipping Kind image load."
  exit 0
fi

echo "Loaded ${#IMAGES[@]} image(s) into ${CLUSTER_NAME}."
