# Event Flow Reference

TradeOps uses Redpanda/Kafka for cross-service domain events and Mosquitto/MQTT for raw market tick ingestion.

## Topic Map

| Topic | Producer | Consumers | Purpose |
| --- | --- | --- | --- |
| `market.ticks` | `market-data-service` | `surveillance-service` | Normalized market tick events. |
| `order.created` | `order-service` | `surveillance-service` | Order submitted/created event. |
| `order.validated` | `order-service` | None currently | Order validation lifecycle event. |
| `order.accepted` | `order-service` | None currently | Accepted order lifecycle event. |
| `order.filled` | `order-service` | `portfolio-service`, `surveillance-service` | Filled order event. |
| `order.rejected` | `order-service` | None currently | Rejected order lifecycle event. |
| `order.cancelled` | `order-service` | `surveillance-service` | Cancelled order event. |
| `portfolio.updated` | `portfolio-service` | `surveillance-service` | Holdings/cash update event. |
| `portfolio.snapshot.created` | `portfolio-service` | None currently | Portfolio snapshot event. |
| `strategy.signal.generated` | `strategy-service` | `surveillance-service` | Strategy signal event. |
| `strategy.backtest.completed` | `strategy-service` | None currently | Backtest completion event. |
| `risk.score.updated` | `risk-engine-service` | `surveillance-service` | Risk score update event. |
| `risk.breached` | `risk-engine-service` | None currently | Risk threshold breach event. |
| `risk.anomaly.detected` | `risk-engine-service` | None currently | Risk anomaly event. |
| `surveillance.alert.created` | `surveillance-service` | `notification-service` | New surveillance alert event. |
| `surveillance.alert.acknowledged` | `surveillance-service` | `notification-service` | Alert acknowledged event. |
| `surveillance.alert.resolved` | `surveillance-service` | `notification-service` | Alert resolved event. |
| `surveillance.alert.dismissed` | `surveillance-service` | `notification-service` | Alert dismissed event. |
| `notification.created` | `notification-service` | None currently | Notification created event. |
| `notification.sent` | `notification-service` | None currently | Notification delivery success event. |
| `notification.failed` | `notification-service` | None currently | Notification delivery failure event. |
| `notification.read` | `notification-service` | None currently | Notification marked read event. |
| `notification.retry_requested` | `notification-service` | None currently | Notification retry requested event. |

## End-To-End Event Story

```mermaid
sequenceDiagram
  participant Client
  participant Gateway as API Gateway
  participant Order as order-service
  participant Kafka as Redpanda
  participant Portfolio as portfolio-service
  participant Surveillance as surveillance-service
  participant Notification as notification-service

  Client->>Gateway: POST /api/orders
  Gateway->>Order: Forward order command
  Order->>Kafka: order.created / order.accepted / order.filled
  Kafka->>Portfolio: order.filled
  Portfolio->>Kafka: portfolio.updated
  Kafka->>Surveillance: order.created / order.filled / portfolio.updated
  Surveillance->>Kafka: surveillance.alert.created
  Kafka->>Notification: surveillance.alert.created
  Notification->>Kafka: notification.created / notification.sent
  Client->>Gateway: GET /api/surveillance/alerts
  Client->>Gateway: GET /api/notifications
```

## Demo Payloads

- Surveillance payloads live under `docs/examples/surveillance/`.
- Notification payloads live under `docs/examples/notifications/`.
- Demo scripts publish compact JSON to Redpanda with `rpk topic produce`.

## Current Limitations

- Event schemas are documented by example payloads, not enforced by schema registry.
- Some published topics are intentionally not consumed yet.
- `portfolio.updated` and `strategy.signal.generated` are consumed by surveillance, but not all consumed events currently trigger rules.
- Notification lifecycle events are published for observability and future integration, but no downstream service consumes them yet.
