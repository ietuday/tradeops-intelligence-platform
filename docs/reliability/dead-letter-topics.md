# Dead-Letter Topics

Dead-letter topics capture source events that could not be processed after retry exhaustion. They are intended for debugging, manual inspection, and controlled replay.

## Topics

| Service | DLQ topic | Source examples |
| --- | --- | --- |
| `portfolio-service` | `portfolio.dlq` | `order.filled` |
| `surveillance-service` | `surveillance.dlq` | `order.created`, `order.cancelled`, `risk.score.updated`, `market.ticks` |
| `notification-service` | `notification.dlq` | `surveillance.alert.created`, `surveillance.alert.acknowledged`, `surveillance.alert.resolved`, `surveillance.alert.dismissed` |
| `audit-service` | `audit.dlq` | User, order, portfolio, risk, surveillance, and notification audit source events |

## Payload Shape

```json
{
  "originalTopic": "surveillance.alert.created",
  "originalPayload": "{\"bad\":true}",
  "errorMessage": "invalid surveillance alert event",
  "serviceName": "notification-service",
  "failedAt": "2026-06-08T10:30:00Z",
  "correlationId": "demo-correlation-id",
  "retryCount": 3
}
```

## Inspect DLQ Messages

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume portfolio.dlq -n 1
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume surveillance.dlq -n 1
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume notification.dlq -n 1
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume audit.dlq -n 1
```

## Manual Replay

Replay should be deliberate. Inspect the `errorMessage`, fix the payload or service condition, then publish the `originalPayload` to `originalTopic`.

Example:

```bash
printf '%s\n' '<originalPayload>' | \
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce '<originalTopic>'
```

Do not bulk replay DLQ topics without checking idempotency and understanding why the events failed.
