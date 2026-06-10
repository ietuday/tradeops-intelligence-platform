# Event Catalog

| Topic | Producer | Consumers | Schema file | Version | Description |
| --- | --- | --- | --- | --- | --- |
| `market.ticks` | `market-data-service` | `surveillance-service`, API Gateway WebSocket | `schemas/events/market/market.ticks.v1.json` | v1 | Normalized market tick event. |
| `order.created` | `order-service` | `surveillance-service`, `audit-service`, API Gateway WebSocket | `schemas/events/orders/order.created.v1.json` | v1 | Order submitted/created event. |
| `order.validated` | `order-service` | API Gateway WebSocket | `schemas/events/orders/order.validated.v1.json` | v1 | Order validation lifecycle event. |
| `order.accepted` | `order-service` | API Gateway WebSocket | `schemas/events/orders/order.accepted.v1.json` | v1 | Accepted order lifecycle event. |
| `order.filled` | `order-service` | `portfolio-service`, `surveillance-service`, `audit-service`, API Gateway WebSocket | `schemas/events/orders/order.filled.v1.json` | v1 | Filled order event. |
| `order.rejected` | `order-service` | API Gateway WebSocket | `schemas/events/orders/order.rejected.v1.json` | v1 | Rejected order lifecycle event. |
| `order.cancelled` | `order-service` | `surveillance-service`, `audit-service`, API Gateway WebSocket | `schemas/events/orders/order.cancelled.v1.json` | v1 | Cancelled order event. |
| `portfolio.updated` | `portfolio-service` | `surveillance-service`, `audit-service` | `schemas/events/portfolio/portfolio.updated.v1.json` | v1 | Portfolio holdings/cash update event. |
| `portfolio.snapshot.created` | `portfolio-service` | None currently | `schemas/events/portfolio/portfolio.snapshot.created.v1.json` | v1 | Portfolio snapshot event. |
| `strategy.signal.generated` | `strategy-service` | `surveillance-service` | `schemas/events/strategy/strategy.signal.generated.v1.json` | v1 | Strategy signal event. |
| `strategy.backtest.completed` | `strategy-service` | None currently | `schemas/events/strategy/strategy.backtest.completed.v1.json` | v1 | Backtest completion event. |
| `risk.score.updated` | `risk-engine-service` | `surveillance-service`, `audit-service` | `schemas/events/risk/risk.score.updated.v1.json` | v1 | Risk score update event. |
| `risk.breached` | `risk-engine-service` | `audit-service` | `schemas/events/risk/risk.breached.v1.json` | v1 | Risk threshold breach event. |
| `risk.anomaly.detected` | `risk-engine-service` | None currently | `schemas/events/risk/risk.anomaly.detected.v1.json` | v1 | Risk anomaly event. |
| `risk.recommendation.created` | `risk-engine-service` | None currently | `schemas/events/risk/risk.recommendation.created.v1.json` | v1 | Risk recommendation event. |
| `surveillance.alert.created` | `surveillance-service` | `notification-service`, `audit-service`, API Gateway WebSocket | `schemas/events/surveillance/surveillance.alert.created.v1.json` | v1 | New surveillance alert event. |
| `surveillance.alert.acknowledged` | `surveillance-service` | `notification-service`, `audit-service`, API Gateway WebSocket | `schemas/events/surveillance/surveillance.alert.acknowledged.v1.json` | v1 | Alert acknowledged event. |
| `surveillance.alert.resolved` | `surveillance-service` | `notification-service`, `audit-service`, API Gateway WebSocket | `schemas/events/surveillance/surveillance.alert.resolved.v1.json` | v1 | Alert resolved event. |
| `surveillance.alert.dismissed` | `surveillance-service` | `notification-service`, `audit-service`, API Gateway WebSocket | `schemas/events/surveillance/surveillance.alert.dismissed.v1.json` | v1 | Alert dismissed event. |
| `surveillance.rule_config.updated` | `surveillance-service` | Future audit/compliance integrations | `schemas/events/surveillance/surveillance.rule_config.updated.v1.json` | v1 | Surveillance rule config changed event. |
| `surveillance.rule_config.enabled` | `surveillance-service` | Future audit/compliance integrations | `schemas/events/surveillance/surveillance.rule_config.enabled.v1.json` | v1 | Surveillance rule enabled event. |
| `surveillance.rule_config.disabled` | `surveillance-service` | Future audit/compliance integrations | `schemas/events/surveillance/surveillance.rule_config.disabled.v1.json` | v1 | Surveillance rule disabled event. |
| `notification.created` | `notification-service` | API Gateway WebSocket | `schemas/events/notifications/notification.created.v1.json` | v1 | Notification created event. |
| `notification.sent` | `notification-service` | `audit-service`, API Gateway WebSocket | `schemas/events/notifications/notification.sent.v1.json` | v1 | Notification delivery success event. |
| `notification.failed` | `notification-service` | `audit-service`, API Gateway WebSocket | `schemas/events/notifications/notification.failed.v1.json` | v1 | Notification delivery failure event. |
| `notification.read` | `notification-service` | `audit-service`, API Gateway WebSocket | `schemas/events/notifications/notification.read.v1.json` | v1 | Notification marked read event. |
| `notification.retry_requested` | `notification-service` | `audit-service`, API Gateway WebSocket | `schemas/events/notifications/notification.retry_requested.v1.json` | v1 | Notification retry requested event. |
| `audit.log.created` | `audit-service` | API Gateway WebSocket | `schemas/events/audit/audit.log.created.v1.json` | v1 | Audit log created event. |
| `portfolio.dlq` | `portfolio-service` | Manual replay tooling | `schemas/events/common/dlq-message.v1.json` | v1 | Failed portfolio consumer payload. |
| `surveillance.dlq` | `surveillance-service` | Manual replay tooling | `schemas/events/common/dlq-message.v1.json` | v1 | Failed surveillance consumer payload. |
| `notification.dlq` | `notification-service` | Manual replay tooling | `schemas/events/common/dlq-message.v1.json` | v1 | Failed notification consumer payload. |
| `audit.dlq` | `audit-service` | Manual replay tooling | `schemas/events/common/dlq-message.v1.json` | v1 | Failed audit consumer payload. |
| WebSocket stream message | API Gateway | WebSocket clients | `schemas/events/common/websocket-stream-message.v1.json` | v1 | Normalized real-time stream envelope. |
