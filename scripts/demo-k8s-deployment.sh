#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${ROOT_DIR}/scripts/k8s-cluster-create.sh"
"${ROOT_DIR}/scripts/k8s-validate.sh"
"${ROOT_DIR}/scripts/k8s-deploy.sh"
"${ROOT_DIR}/scripts/k8s-smoke-test.sh"
"${ROOT_DIR}/scripts/k8s-status.sh"

cat <<'EOF'
TradeOps Kubernetes demo completed.

Access with port-forwarding:
  kubectl -n tradeops port-forward svc/api-gateway 8080:8080
  kubectl -n tradeops port-forward svc/trading-dashboard-react 4300:8080
  kubectl -n tradeops port-forward svc/shell-angular 4200:8080
EOF

