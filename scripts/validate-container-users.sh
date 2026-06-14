#!/usr/bin/env bash
set -Eeuo pipefail

IMAGES=(
  tradeops/api-gateway:0.1.0
  tradeops/audit-service:1.3.0
  tradeops/identity-service:0.2.0
  tradeops/market-data-service:0.3.0
  tradeops/notification-service:0.9.0
  tradeops/order-service:0.4.0
  tradeops/portfolio-service:0.5.0
  tradeops/risk-engine-service:0.7.0
  tradeops/strategy-service:0.6.0
  tradeops/surveillance-service:0.8.0
)

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is required." >&2
  exit 1
fi

status=0
for image in "${IMAGES[@]}"; do
  if ! docker image inspect "${image}" >/dev/null 2>&1; then
    echo "SKIP: ${image} is not built locally."
    continue
  fi

  user="$(docker inspect "${image}" --format '{{.Config.User}}')"
  printf '%-60s %s\n' "${image}" "${user:-<empty>}"

  if ! [[ "${user}" =~ ^[1-9][0-9]*:[1-9][0-9]*$ ]]; then
    echo "ERROR: ${image} must declare a numeric non-root USER, for example 10001:10001." >&2
    status=1
  fi
done

exit "${status}"
