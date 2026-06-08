# Graceful Shutdown

TradeOps services are expected to stop cleanly during local Docker Compose restarts and production-style rolling restarts.

## HTTP Services

Go services use an interrupt-aware server lifecycle:

- `SIGINT` or `SIGTERM` starts shutdown.
- The HTTP server stops accepting new requests.
- In-flight requests are given a short grace period to finish.
- Database, Kafka consumer, and Kafka producer resources are closed before process exit.

Node.js API Gateway shutdown follows the same intent: stop accepting traffic, let active requests finish, and exit cleanly after the process receives a termination signal.

## Event Consumers

Event-consuming services use context cancellation to stop Kafka loops:

- `portfolio-service` consumes `order.filled`.
- `surveillance-service` consumes order, risk, market, portfolio, and strategy events.
- `notification-service` consumes surveillance alert lifecycle events.

On shutdown, consumers stop fetching messages, close readers, and close DLQ writers. Messages are committed only after processing finishes or after the failed event has been retried and sent to a DLQ.

## Local Verification

```bash
docker compose -f infrastructure/docker/docker-compose.yml up -d
docker compose -f infrastructure/docker/docker-compose.yml restart surveillance-service notification-service portfolio-service
docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service notification-service portfolio-service
```

Look for normal shutdown/startup logs and healthy `/health` and `/ready` endpoints after restart.

## Interview Notes

The important reliability behavior is not that shutdown is complex. It is that consumers do not silently drop malformed or failed events during restarts: the retry and DLQ path owns processing failures, while graceful shutdown owns process lifecycle.
