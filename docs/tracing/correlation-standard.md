# Correlation ID Standard

Correlation IDs make it possible to follow one request or event flow through the gateway, services, Kafka events, logs, DLQ messages, and audit logs. v2.3.0 also adds OpenTelemetry trace IDs for Jaeger visualization; both IDs are useful and intentionally coexist.

## Standard Names

| Surface | Standard |
| --- | --- |
| HTTP header | `X-Correlation-ID` |
| JSON event field | `correlationId` |
| Log field | `correlationId` |
| Audit column | `audit_logs.correlation_id` |
| OpenTelemetry span attribute | `correlation.id` |

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
- OpenTelemetry spans include `correlation.id` when available.

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

## Correlation ID Vs Trace ID

- `correlationId` is the stable app-level debug and audit lookup ID.
- `traceId` is the OpenTelemetry trace tree ID shown in Jaeger.
- Audit logs persist `correlationId`; trace IDs may appear in events/spans but are not the primary audit key.
- Do not use `correlationId`, `traceId`, `spanId`, or unbounded `tenantId` as Prometheus labels.

## Known Limitations

- Correlation IDs identify related logs/events; OpenTelemetry adds spans, timing trees, and sampling.
- Some legacy demo payloads may use static IDs for repeatability.

## Future Improvement

Future work can expand tracing to Python services, database spans, webhook spans, and full Kafka header instrumentation.
