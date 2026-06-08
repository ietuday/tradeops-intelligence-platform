#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/infrastructure/docker/docker-compose.yml}"
REDPANDA_SERVICE="${REDPANDA_SERVICE:-redpanda}"
topic=""
confirm="false"
dry_run="true"

usage() {
  cat <<EOF
Usage:
  ./scripts/replay-dlq-events.sh --topic <portfolio.dlq|surveillance.dlq|notification.dlq|audit.dlq> [--dry-run]
  ./scripts/replay-dlq-events.sh --topic <portfolio.dlq|surveillance.dlq|notification.dlq|audit.dlq> --confirm

Default mode is dry-run. This script does not bulk replay DLQ messages automatically.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --topic)
      topic="${2:-}"
      shift 2
      ;;
    --dry-run)
      dry_run="true"
      shift
      ;;
    --confirm)
      confirm="true"
      dry_run="false"
      shift
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

case "${topic}" in
  portfolio.dlq|surveillance.dlq|notification.dlq|audit.dlq) ;;
  *) usage; exit 1 ;;
esac

echo "TradeOps DLQ replay helper"
echo "Topic: ${topic}"
echo "Root cause should be fixed before replay."
echo

cat <<EOF
Inspect one DLQ message:
docker compose -f infrastructure/docker/docker-compose.yml exec ${REDPANDA_SERVICE} rpk topic consume ${topic} -n 1

Inspect more messages:
docker compose -f infrastructure/docker/docker-compose.yml exec ${REDPANDA_SERVICE} rpk topic consume ${topic} -n 5
EOF

if [ "${dry_run}" = "true" ]; then
  cat <<'EOF'

Dry-run mode: no replay attempted.

Replay process:
1. Inspect errorMessage.
2. Fix the bad payload or failing dependency.
3. Extract originalTopic and originalPayload.
4. Publish originalPayload back to originalTopic.

Manual replay shape:
printf '%s\n' '<originalPayload>' | \
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce '<originalTopic>'
EOF
  exit 0
fi

if [ "${confirm}" = "true" ]; then
  cat <<'EOF'

Confirmation received, but automatic bulk replay is intentionally not implemented.
Use this exact manual command after inspecting a single DLQ message:

printf '%s\n' '<originalPayload>' | \
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce '<originalTopic>'

This conservative behavior prevents replaying unknown failed messages blindly.
EOF
fi

