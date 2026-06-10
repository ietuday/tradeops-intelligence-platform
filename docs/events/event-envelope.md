# Event Envelope

TradeOps is moving toward a shared event envelope for platform events. Existing producers may still emit flat payloads, so v2.5.0 treats envelope metadata as recommended and backward-compatible rather than mandatory.

## Standard Shape

```json
{
  "eventId": "uuid-or-stable-id",
  "eventType": "order.created",
  "eventVersion": "1.0",
  "tenantId": "default-tenant",
  "correlationId": "demo-correlation-123",
  "traceparent": "00-optional-w3c-trace-id-optional-span-id-01",
  "occurredAt": "2026-06-10T10:00:00Z",
  "source": "order-service",
  "payload": {}
}
```

## Metadata

| Field | Status | Purpose |
| --- | --- | --- |
| `eventId` | Recommended | Unique or stable event identifier for idempotency and troubleshooting. |
| `eventType` | Recommended | Topic-aligned event name such as `order.created`. |
| `eventVersion` | Recommended | Contract version, currently `"1.0"` for v1 schemas. |
| `tenantId` | Recommended for tenant-owned events | Tenant context for consumers, audit logs, and WebSocket filtering. |
| `correlationId` | Recommended | Business/debug flow ID propagated across HTTP, events, logs, DLQ, and audit. |
| `traceparent` | Optional | W3C trace context when OpenTelemetry context is available. |
| `occurredAt` | Recommended | Event occurrence timestamp in UTC. |
| `source` | Recommended | Producing service name. |
| `payload` | Future direction | Domain payload wrapper. Existing events may keep domain fields top-level. |

## Compatibility

Current schemas allow flat payloads with optional envelope metadata. Producers should add metadata additively and consumers should ignore unknown fields.

## Examples

Flat compatible payload:

```json
{
  "eventType": "order.created",
  "eventVersion": "1.0",
  "tenantId": "default-tenant",
  "correlationId": "demo-correlation-123",
  "orderId": "order-123",
  "symbol": "AAPL",
  "quantity": 100
}
```

Future wrapped payload:

```json
{
  "eventType": "order.created",
  "eventVersion": "1.0",
  "tenantId": "default-tenant",
  "correlationId": "demo-correlation-123",
  "source": "order-service",
  "payload": {
    "orderId": "order-123",
    "symbol": "AAPL",
    "quantity": 100
  }
}
```
