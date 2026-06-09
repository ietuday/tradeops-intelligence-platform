# Correlation ID Standard

Correlation IDs make it possible to follow one request or event flow through the gateway, services, Kafka events, logs, DLQ messages, and audit logs without adding a heavy tracing stack.

## Standard Names

| Surface | Standard |
| --- | --- |
| HTTP header | `X-Correlation-ID` |
| JSON event field | `correlationId` |
| Log field | `correlationId` |
| Audit column | `audit_logs.correlation_id` |

## Generation Rule

- If an incoming HTTP request has `X-Correlation-ID`, preserve it.
- If the header is missing, generate a new UUID.
- If a consumed event has `correlationId`, preserve it.
- If an event has no correlation ID and a new event must be emitted, generate one.

## Propagation Rule

- API Gateway returns `X-Correlation-ID` in responses and forwards it to downstream services.
- Services include the active correlation ID in produced Kafka/Redpanda events.
- Consumers include correlation ID in logs, retry/DLQ payloads, and outgoing events.
- Audit-service persists `correlationId` or `correlation_id` into `audit_logs.correlation_id`.

## HTTP Example

```bash
curl http://localhost:8080/api/orders/health \
  -H "X-Correlation-ID: demo-correlation-123"
```

Expected response header:

```text
X-Correlation-ID: demo-correlation-123
```

## Kafka Event Example

```json
{
  "eventId": "event-1",
  "eventType": "order.created",
  "orderId": "order-1",
  "userId": "user-1",
  "correlationId": "demo-correlation-123"
}
```

## Known Limitations

- This release does not add Jaeger, Tempo, OpenTelemetry Collector, or Loki.
- Correlation IDs identify related logs/events but do not provide spans, timing trees, or distributed trace sampling.
- Some legacy demo payloads may use static IDs for repeatability.

## Future Improvement

OpenTelemetry tracing can be added later once the platform needs span-level latency attribution across HTTP, Kafka, database, and webhook calls.

