# Event Replay

Event replay is useful for demos, regression checks, and event-consumer troubleshooting. It can also create duplicate side effects when used carelessly, so replay should be deliberate.

## When Replay Is Useful

- Demonstrating surveillance alert creation from a sample order event.
- Demonstrating notification creation from a surveillance alert event.
- Demonstrating audit ingestion from source events.
- Re-running known-good payloads after restarting services.

## When Replay Is Dangerous

- The source event has already been processed and idempotency behavior is unknown.
- A downstream dependency is still failing.
- The payload is malformed or does not match the expected topic.
- The event represents a real destructive business action.

## Sample Replay Script

Replay known sample payloads:

```bash
./scripts/replay-sample-events.sh --surveillance
./scripts/replay-sample-events.sh --notifications
./scripts/replay-sample-events.sh --audit
./scripts/replay-sample-events.sh --all
```

Mappings include:

| File | Topic |
| --- | --- |
| `docs/examples/surveillance/order-created-large-order.json` | `order.created` |
| `docs/examples/notifications/surveillance-alert-created-high.json` | `surveillance.alert.created` |
| `docs/examples/audit/order-created-audit-event.json` | `order.created` |
| `docs/examples/audit/surveillance-alert-created-audit-event.json` | `surveillance.alert.created` |

## Manual Redpanda Commands

```bash
node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync("docs/examples/surveillance/order-created-large-order.json", "utf8"))) + "\n");' | \
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce order.created
```

## Replay Safety Checklist

- Confirm the target topic.
- Confirm the payload is valid JSON.
- Confirm the consumer is healthy.
- Check whether the event has a unique event ID.
- Understand expected idempotency and duplicate handling.
- Watch service logs and Prometheus metrics after replay.

