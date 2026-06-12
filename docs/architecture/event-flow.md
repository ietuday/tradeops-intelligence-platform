# Event Flow Reference

TradeOps uses Redpanda/Kafka for cross-service domain events and Mosquitto/MQTT for raw market tick ingestion.

Events should include `correlationId` when available. API Gateway accepts or generates `X-Correlation-ID`, services propagate it to Kafka events, consumers preserve it in downstream events and DLQ records, and audit-service stores it in `audit_logs.correlation_id`.

Tenant-owned events should also include top-level `tenantId`. Consumers preserve it, audit-service stores it as `tenant_id`, and WebSocket streams use it for tenant filtering. Older demo payloads fall back to `default-tenant`.

v2.3.0 keeps the existing `correlationId` event field and adds lightweight `traceparent` propagation where practical. Consumers can continue processing old events without trace fields; they simply start a new trace while preserving `correlationId`.

v2.5.0 adds repository-local JSON Schemas for core Kafka/Redpanda topics, DLQ payloads, and WebSocket stream messages. The schemas are versioned under `schemas/events/`, keep current flat payloads backward-compatible, and allow optional metadata such as `eventVersion`, `tenantId`, `correlationId`, and `traceparent`.

## Topic Map

| Topic | Producer | Consumers | Purpose |
| --- | --- | --- | --- |
| `user.registered` | `identity-service` | `audit-service` | User registration audit event. |
| `user.login` | `identity-service` | `audit-service` | User login audit event. |
| `user.logout` | `identity-service` | `audit-service` | User logout audit event. |
| `market.ticks` | `market-data-service` | `surveillance-service` | Normalized market tick events. |
| `order.created` | `order-service` | `surveillance-service`, `audit-service` | Order submitted/created event. |
| `order.validated` | `order-service` | None currently | Order validation lifecycle event. |
| `order.accepted` | `order-service` | None currently | Accepted order lifecycle event. |
| `order.filled` | `order-service` | `portfolio-service`, `surveillance-service`, `audit-service` | Filled order event. |
| `order.rejected` | `order-service` | None currently | Rejected order lifecycle event. |
| `order.cancelled` | `order-service` | `surveillance-service`, `audit-service` | Cancelled order event. |
| `portfolio.updated` | `portfolio-service` | `surveillance-service`, `audit-service` | Holdings/cash update event. |
| `portfolio.snapshot.created` | `portfolio-service` | None currently | Portfolio snapshot event. |
| `strategy.signal.generated` | `strategy-service` | `surveillance-service` | Strategy signal event. |
| `strategy.backtest.completed` | `strategy-service` | None currently | Backtest completion event. |
| `risk.score.updated` | `risk-engine-service` | `surveillance-service`, `audit-service` | Risk score update event. |
| `risk.breached` | `risk-engine-service` | `audit-service` | Risk threshold breach event. |
| `risk.anomaly.detected` | `risk-engine-service` | None currently | Risk anomaly event. |
| `risk.stress_test.completed` | `risk-engine-service` | Future audit/compliance integrations | Stress test completed event. |
| `risk.scenario.completed` | `risk-engine-service` | Future audit/compliance integrations | Named scenario analysis completed event. |
| `risk.concentration.analyzed` | `risk-engine-service` | Future audit/compliance integrations | Concentration risk analysis completed event. |
| `risk.drawdown.analyzed` | `risk-engine-service` | Future audit/compliance integrations | Drawdown trend analysis completed event. |
| `surveillance.alert.created` | `surveillance-service` | `notification-service`, `audit-service` | New surveillance alert event. |
| `surveillance.alert.acknowledged` | `surveillance-service` | `notification-service`, `audit-service` | Alert acknowledged event. |
| `surveillance.alert.resolved` | `surveillance-service` | `notification-service`, `audit-service` | Alert resolved event. |
| `surveillance.alert.dismissed` | `surveillance-service` | `notification-service`, `audit-service` | Alert dismissed event. |
| `surveillance.rule_simulation.requested` | `surveillance-service` | Future audit/compliance integrations | Dry-run rule simulation requested event. |
| `surveillance.rule_simulation.completed` | `surveillance-service` | Future audit/compliance integrations | Dry-run rule simulation completed event. |
| `surveillance.rule_simulation.failed` | `surveillance-service` | Future audit/compliance integrations | Dry-run rule simulation failed event. |
| `notification.created` | `notification-service` | None currently | Notification created event. |
| `notification.sent` | `notification-service` | `audit-service` | Notification delivery success event. |
| `notification.failed` | `notification-service` | `audit-service` | Notification delivery failure event. |
| `notification.read` | `notification-service` | `audit-service` | Notification marked read event. |
| `notification.retry_requested` | `notification-service` | `audit-service` | Notification retry requested event. |
| `audit.log.created` | `audit-service` | None currently | Audit log created event. |

The API Gateway WebSocket layer also consumes selected topics for live clients: `market.ticks`, order lifecycle topics, `surveillance.alert.*`, `notification.*`, and `audit.log.created`.

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
  participant Audit as audit-service

  Client->>Gateway: POST /api/orders
  Gateway->>Order: Forward order command
  Order->>Kafka: order.created / order.accepted / order.filled
  Kafka->>Portfolio: order.filled
  Portfolio->>Kafka: portfolio.updated
  Kafka->>Surveillance: order.created / order.filled / portfolio.updated
  Surveillance->>Kafka: surveillance.alert.created
  Kafka->>Notification: surveillance.alert.created
  Notification->>Kafka: notification.created / notification.sent
  Kafka->>Audit: order / portfolio / risk / alert / notification events
  Audit->>Kafka: audit.log.created
  Kafka->>Gateway: streamed topics for /ws/*
  Client->>Gateway: GET /api/surveillance/alerts
  Client->>Gateway: GET /api/notifications
  Client->>Gateway: WebSocket /ws/alerts
```

## Demo Payloads

- Surveillance payloads live under `docs/examples/surveillance/`.
- Rule simulation runs through HTTP APIs and publishes `surveillance.rule_simulation.*` lifecycle events only; it does not publish live `surveillance.alert.*` events.
- Notification payloads live under `docs/examples/notifications/`.
- Audit payloads live under `docs/examples/audit/`.
- Risk analytics payloads live under `docs/examples/risk/`.
- WebSocket stream message examples live under `docs/examples/websocket/`.
- Versioned event schemas live under `schemas/events/`.
- Sample-to-schema mappings live in `schemas/events/sample-mapping.json`.
- Demo scripts publish compact JSON to Redpanda with `rpk topic produce`.
- Replay/demo scripts accept `CORRELATION_ID` to inject a traceable `correlationId`.
- Replay/demo scripts default `TENANT_ID=default-tenant` and preserve `tenantId`.

Validate schema JSON and mapped samples:

```bash
./scripts/validate-event-schemas.sh
```

## Current Limitations

- Event schemas are repository-local JSON Schema files and are not enforced by a live schema registry yet.
- Some published topics are intentionally not consumed yet.
- `portfolio.updated` and `strategy.signal.generated` are consumed by surveillance, but not all consumed events currently trigger rules.
- Notification lifecycle events are consumed by audit-service for compliance-style traceability.
