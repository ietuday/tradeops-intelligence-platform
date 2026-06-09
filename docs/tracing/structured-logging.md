# Structured Logging Guidance

TradeOps uses lightweight structured logging conventions so logs can be searched by correlation ID without adding a log aggregation stack.

## Recommended Fields

| Field | Purpose |
| --- | --- |
| `timestamp` | Log event time. |
| `level` | Log severity. |
| `service` | Service name. |
| `message` | Human-readable message. |
| `correlationId` | Request/event correlation ID. |
| `userId` | User or actor when available. |
| `entityId` | Order, alert, notification, audit log, or related entity. |
| `topic` | Kafka topic when processing events. |
| `error` | Error details. |

## Examples

Go:

```go
logger.Info("processed event", "service", "surveillance-service", "topic", topic, "correlationId", correlationID)
```

Node:

```ts
req.log.info({ correlationId: req.headers["x-correlation-id"] }, "proxy request completed");
```

Python:

```python
logger.info("risk event published", extra={"correlationId": correlation_id, "service": "risk-engine-service"})
```

## Grep Logs By Correlation ID

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway order-service surveillance-service notification-service audit-service | \
  grep demo-correlation-123
```

For focused event tracing:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service notification-service audit-service | \
  grep demo-correlation-123
```

