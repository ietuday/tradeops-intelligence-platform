```markdown
# Changelog

All notable changes to TradeOps Intelligence Platform will be documented in this file.

## [v3.1.2] - Idempotent Portfolio Trade Execution Processing

### Added

- Portfolio Service now applies `trade.executed` events atomically to buyer and seller portfolios.
- Added processed-execution idempotency records, payload conflict detection, and a portfolio transaction ledger.
- Extended `trade.executed.v1` additively with execution and ownership fields required for portfolio mutation.

### Changed

- `order.filled` no longer mutates Portfolio Service financial state.
- Portfolio trade consumer defaults to `trade.executed` with a dedicated consumer group and DLQ topic.

### Known Limitations

- Fees are zero, USD-only processing is enforced, short selling and negative cash are disabled, and Portfolio Service still publishes update events directly after commit.

## [v3.1.1] - Transactional Outbox Publisher

### Added

- Background Order Service outbox publisher with lease-based multi-replica claiming, retry backoff, terminal failures, and graceful shutdown.
- Additive `009` migration for `locked_by`, `locked_at`, `lease_until`, and `failed_at` on `order_outbox`.
- Kafka publication from stored outbox payloads with stable `tenantID:aggregateID` keys and correlation/trace headers.
- Prometheus `tradeops_outbox_*` metrics and `GET /internal/outbox/status` operational visibility.
- Compose and Helm configuration for publisher polling, timeout, lease, backoff, attempts, and error length settings.
- Architecture, runbook, and consumer idempotency documentation.

### Changed

- Order Service no longer publishes order events inline after DB commit; the outbox publisher owns Kafka delivery.

### Delivery Semantics

- Database state and event intent are committed atomically. Kafka delivery is at-least-once. Consumers must be idempotent.

## [v3.1.0] - Smart Order Execution and OMS Enhancement

### Added

- Deterministic Order Service matching engine with price-time priority, partial fills, IOC, and FOK behavior.
- Additive order-management migration for filled/remaining quantities, average fill price, time-in-force, optimistic versions, executions, transactional outbox, and risk decisions.
- New Order Service APIs for amendment, executions, trades, and aggregated order-book depth.
- API Gateway proxy support for OMS amendment, trade, execution, and book-depth endpoints.
- Event schemas for `order.partially_filled`, `order.amended`, `order.expired`, and `trade.executed`.
- OMS architecture, matching, lifecycle, outbox, and runbook documentation.

### Changed

- Order creation no longer uses simulated market fill prices; market orders match available resting liquidity and do not rest remainders.
- Cancellation now supports partially filled orders and clears remaining quantity.
- `STOP_LOSS` remains accepted as a backward-compatible alias for `STOP`.

### Known Limitations

- Pre-trade remote risk evaluation, stop-trigger workers, expiry workers, WebSocket book streaming, and portfolio `trade.executed` consumption are documented/scaffolded but not fully implemented in this slice.
- Kafka/outbox delivery is at-least-once; consumers must remain idempotent.

## [v0.1.0] - Platform Foundation

### Added

- Initial monorepo structure
- API Gateway foundation
- Docker Compose foundation
- PostgreSQL
- Redis
- Mosquitto MQTT
- Redpanda Kafka-compatible event bus
- Prometheus
- Grafana
- Angular shell placeholder
- React trading dashboard placeholder
- Smoke test script
- Release notes
Step 8: Add Release Notes

File:

docs/release-notes/v0.1.0.md

Content:

# v0.1.0 - Platform Foundation

## Added

- Initial monorepo structure
- API Gateway service foundation
- Angular shell placeholder
- React trading dashboard placeholder
- Docker Compose setup
- PostgreSQL container
- Redis container
- Mosquitto MQTT broker
- Redpanda Kafka-compatible event bus
- Prometheus metrics stack
- Grafana dashboard stack
- Makefile commands
- Smoke test script

## Infrastructure

- PostgreSQL exposed on port `5432`
- Redis exposed on port `6379`
- Mosquitto exposed on port `1883`
- Redpanda Kafka exposed on port `9092`
- Prometheus exposed on port `9090`
- Grafana exposed on port `3000`
- API Gateway exposed on port `8080`

## Testing

- Smoke test validates API Gateway `/health`, `/ready`, and `/metrics`

## Known Limitations

- Authentication is not implemented yet
- Market data pipeline is not implemented yet
- Trading order flow is not implemented yet
- AI assistant is not implemented yet
- Kubernetes deployment is planned for a later release

## Next Release

v0.2.0 will add Identity and RBAC.
