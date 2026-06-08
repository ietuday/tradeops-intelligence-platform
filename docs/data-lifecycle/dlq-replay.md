# DLQ Replay

Dead-letter queues capture events that failed all retry attempts. DLQ replay should happen only after the root cause is understood and fixed.

## DLQ Topics

| Service | DLQ Topic |
| --- | --- |
| `portfolio-service` | `portfolio.dlq` |
| `surveillance-service` | `surveillance.dlq` |
| `notification-service` | `notification.dlq` |
| `audit-service` | `audit.dlq` |

## Payload Format

Typical DLQ records include:

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

## Inspect Messages

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume surveillance.dlq -n 1
```

## Extract And Replay

1. Inspect `errorMessage`.
2. Fix the bad payload or failing dependency.
3. Extract `originalPayload`.
4. Publish `originalPayload` to `originalTopic`.

Manual replay shape:

```bash
printf '%s\n' '<originalPayload>' | \
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce '<originalTopic>'
```

The helper script is conservative:

```bash
./scripts/replay-dlq-events.sh --topic surveillance.dlq --dry-run
./scripts/replay-dlq-events.sh --topic surveillance.dlq --confirm
```

With `--confirm`, the script still prints exact manual commands instead of bulk replaying unknown messages.

## When Not To Replay

- The root cause is not fixed.
- The payload would duplicate a side effect that is not idempotent.
- The event is too old to be meaningful.
- The target service is unhealthy.
- The message contains sensitive data you are not authorized to handle.

## Idempotency Notes

Portfolio, surveillance, notification, and audit consumers include duplicate handling in key paths, but replay safety depends on the event shape and service behavior. Treat duplicate handling as a safety net, not permission to replay blindly.

