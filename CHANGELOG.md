```markdown
# Changelog

All notable changes to TradeOps Intelligence Platform will be documented in this file.

## [v3.5.0] - Real OAuth2/OIDC Authentication, JWT Validation, and Service Authorization

### Added

- API Gateway JWKS-based RS256 JWT validation, normalized identity headers, route-level role/scope policy, and bounded auth metrics.
- Identity Service local OIDC discovery, JWKS, and token endpoints with ephemeral dev signing key fallback.
- Service-auth validation for trusted gateway headers in Order, Portfolio, and Audit services.
- Auth decision event schema, Docker Compose and Helm auth configuration, docs, and local OIDC/RBAC demo script.

### Known Limitations

- Local service auth uses a shared secret for development; production should use mTLS, workload identity, or a service mesh.
- Auth decision emission is schema-ready, but high-volume audit publishing is intentionally not enabled by default.

## [v3.4.0] - Consumer Lag Monitoring, DLQ Visibility, and Autoscaling Signals

### Added

- Portfolio Service `/internal/consumers/status` endpoint for consumer and DLQ health.
- Shared consumer observability metrics for lag, processing attempts/errors, DLQ counts/age, and outbox backlog.
- Compose and Helm configuration for consumer observability thresholds.
- Disabled-by-default Helm autoscaling values and HPA custom metric rendering for consumer lag.
- Prometheus alert rules, operations docs, release notes, and demo script for lag/DLQ troubleshooting.

### Known Limitations

- Portfolio Service is wired first; other Go consumer services can reuse the metric/status conventions in later releases.
- DLQ visibility is read-only and does not replay or delete messages.

## [v3.3.0] - Portfolio Service Transactional Outbox and Idempotent Event Processing

### Added

- Portfolio Service transactional outbox table and background publisher for reliable `portfolio.updated` publication.
- Tenant-scoped processed-event idempotency keys for `trade.executed` consumption.
- Same-transaction portfolio mutation, processed-event insert, snapshot creation, and outbox insert.
- Portfolio outbox and consumer status endpoints under `/internal/portfolio/*`.
- Portfolio consumer/outbox/idempotency metrics and demo script.

### Known Limitations

- `portfolio.snapshot.created` remains a database snapshot record; v3.3.0 focuses Kafka outbox publication on `portfolio.updated`.
- Portfolio outbox delivery is at-least-once; downstream consumers must remain idempotent.

## [v3.2.0] - Stop Order Triggering and Reference Price Support

### Added

- Order Service `reference_prices` table and market price consumer for latest tenant/symbol prices.
- Stop-trigger worker that activates eligible `STOP` orders as `MARKET` and `STOP_LIMIT` orders as `LIMIT`.
- Idempotent trigger tracking with `triggered_at`, original/activated order type metadata, row locking, and transactional `order.triggered` outbox events.
- Prometheus stop-trigger/reference-price metrics and `GET /internal/stop-trigger/status`.
- Event schemas for `market.price.updated` and `order.triggered`.
- Stop-order triggering docs and demo script.

### Known Limitations

- Triggered orders use their original placement-time pre-trade risk decision.
- Reference prices are latest snapshots, not venue-specific NBBO marks.

## [v3.1.4] - Remote Pre-Trade Risk Evaluation

### Added

- Order Service remote pre-trade risk workflow: persist `risk_pending`, call Risk Engine outside the DB transaction, then atomically approve/match or reject.
- Risk Engine `POST /api/v1/risk/pre-trade/evaluate` with tenant/default policy lookup, max quantity/notional, allowed/restricted symbols, and daily approved-notional checks.
- Additive `012` migration for pre-trade risk policies, richer order risk metadata, and final decision ledger constraints/indexes.
- `order.risk_rejected` transactional outbox event schema and bounded pre-trade metrics on both services.
- Compose/Helm configuration for fail-closed risk evaluation, low retries, timeout, and development-only fail-open.

### Known Limitations

- Market and stop-only orders fail closed with `REFERENCE_PRICE_UNAVAILABLE` until a reliable reference price source is wired into the synchronous order path.

## [v3.1.3] - Order Expiry Worker for DAY and GTD Orders

### Added

- Order Service expiry worker for due `DAY` and `GTD` orders using PostgreSQL `FOR UPDATE SKIP LOCKED` for multi-replica-safe bounded batches.
- Weekday trading calendar abstraction with configurable IANA timezone and close time for server-assigned `DAY` expiries.
- Additive `011` migration for expiry metadata and expiry worker scan index.
- Transactional `order.expired` history/outbox events with stable expiry reason codes.
- Prometheus expiry metrics, local alerts, Compose/Helm configuration, and `GET /internal/order-expiry/status`.

### Known Limitations

- The initial `DAY` calendar is weekday-only and does not model exchange holidays, special sessions, or half-days.

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
