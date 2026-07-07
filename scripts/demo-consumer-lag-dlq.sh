#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
PORTFOLIO_URL="${PORTFOLIO_URL:-http://localhost:8087}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
TOPIC="${PORTFOLIO_TRADE_TOPIC:-trade.executed}"

log() { printf '[consumer-lag-dlq-demo] %s\n' "$*"; }
warn() { printf '[consumer-lag-dlq-demo] WARN: %s\n' "$*" >&2; }

require() {
  command -v "$1" >/dev/null 2>&1 || {
    warn "$1 is not installed; skipping command that needs it"
    return 1
  }
}

query_prometheus() {
  local query="$1"
  local encoded
  encoded="$(printf '%s' "${query}" | sed 's/ /%20/g; s/{/%7B/g; s/}/%7D/g; s/"/%22/g; s/,/%2C/g; s/=/%3D/g; s/>/%3E/g; s/(/%28/g; s/)/%29/g; s/\\[/%5B/g; s/\\]/%5D/g')"
  curl -fsS "${PROMETHEUS_URL}/api/v1/query?query=${encoded}" || true
  printf '\n'
}

log "checking Portfolio Service health at ${PORTFOLIO_URL}"
curl -fsS "${PORTFOLIO_URL}/health" >/dev/null

log "current internal consumer status"
curl -fsS "${PORTFOLIO_URL}/internal/consumers/status"
printf '\n'

log "consumer and DLQ metrics exposed by Portfolio Service"
curl -fsS "${PORTFOLIO_URL}/metrics" | grep -E 'tradeops_(consumer|dlq|outbox)' || true

log "Prometheus lag query"
query_prometheus 'sum by (service, consumer_group, topic) (tradeops_consumer_lag_messages)'

log "Prometheus DLQ query"
query_prometheus 'sum by (service, topic) (tradeops_dlq_messages)'

log "To demonstrate lag growth locally, stop the Portfolio consumer, publish trade.executed events, then start it again:"
cat <<STEPS
docker compose -f "${COMPOSE_FILE}" stop portfolio-service
for i in \$(seq 1 20); do
  printf '{"eventId":"evt_lag_demo_%s","eventType":"trade.executed","eventVersion":"1.0","tenantId":"default-tenant","occurredAt":"%s","executionId":"exec_lag_demo_%s","buyOrderId":"buy-demo","sellOrderId":"sell-demo","buyerUserId":"buyer","sellerUserId":"seller","symbol":"AAPL","executionQuantity":1,"executionPrice":100,"currency":"USD","source":"order-service"}\n' "\$i" "\$(date -u +"%Y-%m-%dT%H:%M:%SZ")" "\$i" |
    docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${TOPIC}" >/dev/null
done
docker compose -f "${COMPOSE_FILE}" start portfolio-service
curl -fsS "${PORTFOLIO_URL}/internal/consumers/status"
STEPS

if require docker; then
  log "publishing one intentionally invalid event to exercise retry/DLQ visibility"
  printf '{"eventId":"evt_dlq_demo","eventType":"trade.executed","eventVersion":"1.0","tenantId":"default-tenant","occurredAt":"%s","executionId":"exec_dlq_demo","symbol":"AAPL","executionQuantity":-1,"executionPrice":100,"currency":"USD","source":"demo"}\n' "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" |
    docker compose -f "${COMPOSE_FILE}" exec -T redpanda rpk topic produce "${TOPIC}" >/dev/null || warn "could not publish invalid event; is the Compose stack running?"
  sleep 3
  log "status after invalid event"
  curl -fsS "${PORTFOLIO_URL}/internal/consumers/status" || true
  printf '\n'
  log "DLQ metrics after invalid event"
  curl -fsS "${PORTFOLIO_URL}/metrics" | grep -E 'tradeops_dlq|tradeops_consumer_processing_errors' || true
fi

log "Troubleshooting hints:"
log "- Lag with low errors usually means scale consumers or inspect downstream latency."
log "- Lag with processing errors means fix the payload/schema/dependency failure first."
log "- DLQ visibility is read-only in v3.4.0; use approved replay tooling for recovery."
