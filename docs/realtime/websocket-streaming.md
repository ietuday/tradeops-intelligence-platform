# WebSocket Streaming

TradeOps exposes lightweight real-time WebSocket streams through the existing API Gateway so clients can receive live platform events without polling.

## Endpoints

| Endpoint | Stream | Topics |
| --- | --- | --- |
| `/ws` | All allowed events | All mapped topics. |
| `/ws/market` | Market ticks | `market.ticks` |
| `/ws/orders` | Order lifecycle | `order.created`, `order.validated`, `order.accepted`, `order.filled`, `order.rejected`, `order.cancelled` |
| `/ws/alerts` | Surveillance alerts | `surveillance.alert.created`, `surveillance.alert.acknowledged`, `surveillance.alert.resolved` |
| `/ws/notifications` | Notifications | `notification.created`, `notification.sent`, `notification.failed`, `notification.read` |
| `/ws/audit` | Audit events | `audit.log.created` |

## Configuration

```text
WS_ENABLED=true
WS_REQUIRE_AUTH=true
WS_KAFKA_BROKERS=redpanda:29092
WS_KAFKA_GROUP_ID=api-gateway-websocket
WS_ALLOWED_ORIGINS=http://localhost:4200,http://localhost:4300
WS_MAX_CONNECTIONS=100
WS_HEARTBEAT_INTERVAL_MS=30000
```

## Auth Model

When `WS_REQUIRE_AUTH=true`, clients pass a JWT either as:

```text
ws://localhost:8080/ws/orders?token=<jwt>
```

or with an `Authorization: Bearer <jwt>` header if the WebSocket client supports upgrade headers.

Role access is intentionally simple:

| Role | Streams |
| --- | --- |
| `trading_admin` | All streams. |
| `risk_manager` | Market, alerts, notifications, audit. |
| `analyst` | Market, alerts, notifications, audit. |
| `trader` | Market, orders, notifications. |
| `viewer` | Market and alerts. |

## Message Format

Event message:

```json
{
  "type": "order.filled",
  "topic": "order.filled",
  "correlationId": "demo-correlation-123",
  "timestamp": "2026-06-09T10:00:00Z",
  "payload": {}
}
```

Connection message:

```json
{
  "type": "connection.ready",
  "stream": "orders",
  "correlationId": "demo-correlation-123",
  "timestamp": "2026-06-09T10:00:00Z"
}
```

Heartbeat message:

```json
{
  "type": "heartbeat",
  "stream": "orders",
  "correlationId": "demo-correlation-123",
  "timestamp": "2026-06-09T10:00:30Z"
}
```

## Correlation IDs

The gateway uses `X-Correlation-ID`, `correlationId` query parameter, or a generated UUID for the connection. Kafka payloads preserve `correlationId` or `correlation_id` when present.

## Tenant Filtering

WebSocket connections inherit `tenantId` from the JWT used during upgrade. Events with `payload.tenantId` are sent only to clients with the same tenant or to `trading_admin` clients. Events without `tenantId` are treated as global compatibility events.

Tenant IDs are not used as metric labels to avoid high-cardinality Prometheus series.

## Metrics

| Metric | Purpose |
| --- | --- |
| `tradeops_api_gateway_websocket_connections_active{stream}` | Current active WebSocket clients. |
| `tradeops_api_gateway_websocket_connections_total{stream}` | Accepted WebSocket clients. |
| `tradeops_api_gateway_websocket_messages_sent_total{stream,topic}` | Messages sent to clients. |
| `tradeops_api_gateway_websocket_messages_failed_total{stream,topic}` | Failed message sends. |
| `tradeops_api_gateway_websocket_auth_failures_total{stream}` | Failed WebSocket auth attempts. |
| `tradeops_api_gateway_websocket_kafka_events_consumed_total{topic}` | Kafka events consumed for streaming. |

## Local Demo

```bash
./scripts/demo-websocket-streams.sh
TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --orders
TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --alerts --publish-sample
```

If `websocat` is unavailable, the script prints manual connection instructions.

## Known Limitations

- The WebSocket consumer runs inside the API Gateway for local/demo simplicity.
- There is no replay buffer; clients receive events after connection.
- Auth is JWT/RBAC based and intentionally lightweight.
- Production deployments should consider dedicated streaming gateways, backpressure policies, and horizontal scaling behavior.
