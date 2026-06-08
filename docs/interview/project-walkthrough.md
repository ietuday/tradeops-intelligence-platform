# Interview Project Walkthrough

## 60-Second Explanation

TradeOps Intelligence Platform is a local microservices trading intelligence system. It has an API Gateway, identity/auth, market data ingestion, order management, portfolio updates, strategy and risk analytics, surveillance alerting, notification delivery, audit trails, Prometheus metrics, Grafana dashboards, and Redpanda/Kafka event flows. The project shows how trading workflows move from synchronous APIs into event-driven processing: orders publish events, portfolio and surveillance consume them, surveillance creates alerts, notifications are created from alert lifecycle events, and audit-service records a compliance-style event trail.

## 2-Minute Explanation

The platform is organized around independent services. The API Gateway is the client entry point. Identity issues JWTs. The order service handles order creation with idempotency. A filled order emits an event that portfolio consumes to update holdings and publish portfolio updates. Risk and strategy services generate analytics and events. Surveillance consumes market, order, portfolio, risk, and strategy events, runs rules such as large order and abnormal price movement detection, and creates alert lifecycle events. Notification consumes surveillance alert events, creates user notifications, supports preferences, and records delivery attempts. Audit consumes important user, business, and system events and stores searchable audit logs.

Operationally, every service exposes health, readiness, and metrics endpoints. Prometheus scrapes the services, Grafana has dashboard exports for platform health, gateway traffic, event processing, surveillance/notifications, and audit/compliance, and Docker Compose runs the full local stack with PostgreSQL, Redis, Mosquitto, and Redpanda.

## Architecture Explanation

The architecture uses a gateway plus service-owned domains. HTTP is used for user-driven queries and commands. Kafka is used for cross-service events that should not require synchronous coupling. PostgreSQL stores durable service data. Redis supports identity refresh/session state. Mosquitto represents raw market tick ingestion, while Redpanda is the durable event backbone.

## Event-Driven Flow Explanation

Market data starts as MQTT ticks, is normalized by the market data service, and is published as `market.ticks`. Orders publish lifecycle topics such as `order.created`, `order.filled`, and `order.cancelled`. Portfolio consumes fills and publishes `portfolio.updated`. Risk publishes `risk.score.updated` and related risk events. Surveillance consumes those topics, applies rules, and publishes `surveillance.alert.*`. Notification consumes the surveillance alert topics and publishes `notification.*` lifecycle events.

Audit consumes key topics across identity, order, portfolio, risk, surveillance, and notification services. It normalizes them into audit log rows, exposes filtering/summary/export APIs, and publishes `audit.log.created`.

## Why Go, Python, And Node Were Used

Go is used for high-throughput transactional services such as identity, market data, orders, portfolio, surveillance, and notifications because it is simple, fast, and strong for concurrent network services.

Python is used for strategy and risk because analytics-heavy services often benefit from Python’s data ecosystem and quick iteration.

Node.js is used for the API Gateway because Express proxy routing and Jest route tests are lightweight and productive for a gateway layer.

## How Idempotency Is Handled

The order service accepts an `Idempotency-Key` header during order creation. Replaying the same request with the same key returns the same order result instead of creating a duplicate order. That is important in trading systems where clients may retry after a network failure.

## How Observability Is Handled

Each service exposes `/health`, `/ready`, and `/metrics`. Prometheus scrapes all backend services through the Docker Compose network and loads local alert rules for availability, gateway errors/latency, event processing failures, DLQ events, notification delivery failures, and audit ingestion failures. Grafana reads Prometheus and includes SLO-oriented dashboards. The gateway propagates correlation IDs so logs and requests can be connected across services.

## How Kafka/Redpanda Is Used

Redpanda is the local Kafka-compatible broker. Services publish domain events after important state changes. Consumers process events asynchronously and defensively handle malformed payloads. The platform currently uses example payloads and documented topics instead of a full schema registry.

## How Surveillance Works

The surveillance service consumes order, market, portfolio, risk, and strategy events. It evaluates rules such as large orders, rapid orders, high cancellations, abnormal price movement, and risk-score breaches. Matching rules create alerts in PostgreSQL and publish `surveillance.alert.created`. Alert APIs support lifecycle transitions from `OPEN` to `ACKNOWLEDGED`, `RESOLVED`, or `DISMISSED`.

## How Notification Delivery Works

The notification service consumes surveillance alert lifecycle topics. It creates user notifications and sends through configured channels. `IN_APP` persists and marks notifications as sent. `EMAIL` is intentionally mock/log-only. `WEBHOOK` posts to a configured URL with timeout and retry behavior, recording delivery attempts and failures.

## How Audit Trail Works

The audit service consumes important platform events, maps them to normalized actions such as `ORDER_CREATED`, `RISK_BREACHED`, and `SURVEILLANCE_ALERT_CREATED`, and stores them in PostgreSQL. It supports filtered log search, summary counts, JSON/CSV export, idempotency by source event key, retry/backoff, and `audit.dlq` for failed event processing.

## How To Demo The Project

1. Start the stack with `make dev-up`.
2. Run `make smoke` to verify service health and core workflows.
3. Run `./scripts/demo-surveillance.sh` to create and transition a surveillance alert.
4. Run `./scripts/demo-notifications.sh` to publish a surveillance alert event, list notifications, and mark one as read.
5. Run `./scripts/demo-audit.sh` to publish a source event, list audit logs, show summary, and export.
6. Run `./scripts/demo-e2e-tradeops.sh` for a guided end-to-end platform story.
7. Run `./scripts/demo-observability.sh` to walk through dashboards, alert rules, and safe Prometheus queries.
8. Open Prometheus at `http://localhost:9090` and Grafana at `http://localhost:3000`.

## Senior-Level Talking Points

- The system separates synchronous command/query APIs from asynchronous domain events.
- Idempotency is handled at order creation, where duplicate side effects matter most.
- Services own their persistence and expose health/readiness/metrics consistently.
- Observability is treated as a first-class platform capability with dashboards, alert rules, SLO docs, and runbooks.
- Event consumers are defensive so bad demo payloads do not crash the process.
- The gateway keeps external routing stable while services retain internal base paths.
- Audit logging demonstrates compliance-style traceability without coupling business services to synchronous audit writes.
- The project includes demo scripts, release notes, architecture docs, troubleshooting, and production-readiness documentation because operational clarity is part of engineering quality.

## Known Limitations And Future Improvements

- This is a local Compose platform, not a production deployment.
- Kafka schemas are documented by examples but not enforced by a schema registry.
- There is no distributed tracing yet.
- Consumer lag metrics are not implemented yet; current event-health views use retry, failure, duplicate, and DLQ metrics.
- Email delivery is mock-only.
- Kubernetes/Helm, CI/CD, managed secrets, TLS ingress, and production database isolation are future work.
