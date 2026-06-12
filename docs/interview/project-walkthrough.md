# Interview Project Walkthrough

## 60-Second Explanation

TradeOps Intelligence Platform is a local microservices trading intelligence system. It has an API Gateway, identity/auth, API security hardening, admin operations APIs, market data ingestion, order management, portfolio updates, strategy and risk analytics, configurable surveillance alerting, notification delivery, audit trails, real-time WebSocket streaming, correlation ID tracing, Prometheus metrics, Grafana dashboards, data lifecycle scripts, optional Helm deployment manifests, event schema governance, and Redpanda/Kafka event flows. The project shows how trading workflows move from synchronous APIs into event-driven processing: orders publish events, portfolio and surveillance consume them, surveillance creates alerts, notifications are created from alert lifecycle events, audit-service records a compliance-style event trail, and the gateway can stream selected topics to WebSocket clients.

## 2-Minute Explanation

The platform is organized around independent services. The API Gateway is the client entry point. Identity issues JWTs. The order service handles order creation with idempotency. A filled order emits an event that portfolio consumes to update holdings and publish portfolio updates. Risk and strategy services generate analytics and events. Surveillance consumes market, order, portfolio, risk, and strategy events, runs rules such as large order and abnormal price movement detection, supports dry-run simulation for proposed rule configs, and creates alert lifecycle events only in live processing. Notification consumes surveillance alert events, creates user notifications, supports preferences, and records delivery attempts. Audit consumes important user, business, and system events and stores searchable audit logs.

Operationally, every service exposes health, readiness, and metrics endpoints. Prometheus scrapes the services, Grafana has dashboard exports for platform health, gateway traffic, event processing, surveillance/notifications, and audit/compliance, Jaeger shows OpenTelemetry traces for the gateway/order/surveillance/notification/audit path, and Docker Compose runs the full local stack with PostgreSQL, Redis, Mosquitto, Redpanda, and Jaeger.

The API Gateway also exposes `/api/admin` for backend-only operations views: aggregated health, service and topic catalogs, DLQ guidance, audit/alert/notification summaries, rule config summary, and safe masked platform config.

Correlation IDs provide lightweight tracing without a full tracing stack: the gateway accepts or generates `X-Correlation-ID`, services propagate `correlationId` into Kafka events, DLQ records keep the same ID, and audit-service stores it for querying.

The repository also includes data lifecycle support: retention policy docs, local PostgreSQL backup/restore scripts, old-data archival exports, sample event replay, and conservative DLQ replay guidance.

For deployment readiness, the repository includes an optional Helm chart with Deployments, Services, ConfigMap/Secret separation, probes, resource limits, and optional ingress. Docker Compose remains the recommended local demo runtime.

For security readiness, the repository includes a STRIDE-style threat model, RBAC matrix, API security guide, secrets-management guidance, security checklist, and a read-only security validation script. The gateway uses Helmet, configurable CORS, request size limits, local rate limiting, and consistent error responses with correlation IDs.

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

Each service exposes `/health`, `/ready`, and `/metrics`. Prometheus scrapes all backend services through the Docker Compose network and loads local alert rules for availability, gateway errors/latency, event processing failures, DLQ events, notification delivery failures, and audit ingestion failures. Grafana reads Prometheus and includes SLO-oriented dashboards. The gateway propagates correlation IDs so logs, events, DLQ records, and audit logs can be connected across services. OpenTelemetry adds Jaeger traces for span-level timing across the primary order-to-alert-to-notification-to-audit flow.

The admin operations APIs add an operator-friendly aggregation layer without adding a new service or UI. They are read-only by default, RBAC-protected, tenant-aware, and degrade gracefully if one downstream summary endpoint is unavailable.

## How Real-Time Streaming Works

The API Gateway attaches a WebSocket server to the existing HTTP server. It exposes `/ws` plus domain streams for market, orders, alerts, notifications, and audit. A lightweight Kafka consumer maps selected topics to streams, normalizes messages, preserves `correlationId`, sends heartbeat messages, and records WebSocket metrics.

## How Security Hardening Is Handled

The platform keeps security practical for a portfolio system. The API Gateway disables `x-powered-by`, applies Helmet headers, supports `CORS_ORIGIN`, enforces `REQUEST_BODY_LIMIT`, applies a generous in-memory rate limit, and returns security errors with `correlationId`. Backend services validate JWTs and enforce RBAC where implemented. Security docs are explicit about current controls and production gaps such as OAuth/OIDC, mTLS, external secret managers, TLS ingress, and WAF.

## How Multitenancy Is Handled

TradeOps uses shared-database multitenancy for interview-friendly simplicity. Tenant-owned tables carry `tenant_id`, JWTs include `tenantId`, API Gateway forwards `X-Tenant-ID`, tenant-owned events include `tenantId`, audit logs persist tenant context, and WebSocket streams filter tenant events by connection tenant. Existing demo data falls back to `default-tenant`; stronger schema/database-per-tenant isolation is documented as future production hardening.

## How Kafka/Redpanda Is Used

Redpanda is the local Kafka-compatible broker. Services publish domain events after important state changes. Consumers process events asynchronously and defensively handle malformed payloads. Events use `correlationId` where available so one workflow can be followed across producers, consumers, DLQ records, and audit logs. The platform uses repository-local JSON Schemas and sample mappings for event governance without adding a live schema registry.

Sample event replay scripts publish known-good payloads for surveillance, notification, and audit demos. DLQ replay is documented as a manual, root-cause-first operation instead of an automatic bulk replay.

## How Data Lifecycle Is Handled

The project documents retention periods for market ticks, order events, portfolio snapshots, risk scores, surveillance alerts, notifications, audit logs, strategy/backtest data, and DLQ messages. Local scripts can create PostgreSQL backups, restore backups with `--confirm`, export old rows to `archives/YYYY-MM-DD/`, and replay sample events. Deletion is disabled by default and requires explicit confirmation.

## How Deployment Readiness Is Handled

Docker Compose runs the complete local platform. The optional Helm chart demonstrates how application services can be packaged for Kubernetes with ClusterIP services, liveness/readiness probes, resource requests/limits, graceful termination, ConfigMap-based configuration, placeholder Kubernetes Secrets, and optional API Gateway ingress.

## How Surveillance Works

The surveillance service consumes order, market, portfolio, risk, and strategy events. It evaluates rules such as large orders, rapid orders, high cancellations, abnormal price movement, and risk-score breaches. Rule thresholds, severity, and enable/disable state are tenant-aware and database-backed, with environment defaults as fallback. Matching rules create alerts in PostgreSQL and publish `surveillance.alert.created`. Alert APIs support lifecycle transitions from `OPEN` to `ACKNOWLEDGED`, `RESOLVED`, or `DISMISSED`.

Rule simulation adds a production-style dry-run workflow. A caller can submit proposed config changes, the service overlays them on tenant-effective configs in memory, evaluates demo/historical-style events with a throwaway rule engine, and returns matched event and would-trigger alert counts. It does not update `surveillance_rule_configs`, mutate the live cache, create alerts, publish live alert events, or trigger notifications.

## How Notification Delivery Works

The notification service consumes surveillance alert lifecycle topics. It creates user notifications and sends through configured channels. `IN_APP` persists and marks notifications as sent. `EMAIL` is intentionally mock/log-only. `WEBHOOK` posts to a configured URL with timeout and retry behavior, recording delivery attempts and failures.

## How Audit Trail Works

The audit service consumes important platform events, maps them to normalized actions such as `ORDER_CREATED`, `RISK_BREACHED`, and `SURVEILLANCE_ALERT_CREATED`, and stores them in PostgreSQL. It supports filtered log search, summary counts, JSON/CSV export, idempotency by source event key, retry/backoff, and `audit.dlq` for failed event processing.

## How To Demo The Project

1. Start the stack with `make dev-up`.
2. Run `make smoke` to verify service health and core workflows.
3. Run `./scripts/demo-surveillance.sh` to create and transition a surveillance alert.
4. Run `TOKEN=<jwt> ./scripts/demo-rule-config.sh` to show tenant-aware rule thresholds and enable/disable APIs.
5. Run `TOKEN=<jwt> ./scripts/demo-rule-simulation.sh` to show a dry-run threshold comparison with no live alert side effects.
6. Run `./scripts/demo-notifications.sh` to publish a surveillance alert event, list notifications, and mark one as read.
7. Run `./scripts/demo-audit.sh` to publish a source event, list audit logs, show summary, and export.
8. Run `TOKEN=<jwt> ./scripts/demo-admin-ops.sh` to show backend admin operations APIs.
9. Run `./scripts/demo-e2e-tradeops.sh` for a guided end-to-end platform story.
10. Run `./scripts/demo-observability.sh` to walk through dashboards, alert rules, and safe Prometheus queries.
11. Run `./scripts/db-backup.sh` and `./scripts/archive-old-data.sh` to show safe data lifecycle operations.
12. Run `./scripts/validate-helm.sh` to show Kubernetes deployment-readiness validation.
13. Run `./scripts/demo-correlation-tracing.sh` to show request/event correlation visibility.
14. Run `TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --alerts` to show live event streaming.
15. Run `./scripts/security-check.sh` to show safe repository security validation.
16. Open Prometheus at `http://localhost:9090` and Grafana at `http://localhost:3000`.

## Senior-Level Talking Points

- The system separates synchronous command/query APIs from asynchronous domain events.
- Idempotency is handled at order creation, where duplicate side effects matter most.
- Services own their persistence and expose health/readiness/metrics consistently.
- Observability is treated as a first-class platform capability with dashboards, alert rules, SLO docs, and runbooks.
- Correlation IDs make request/event tracing possible without adding a heavy tracing stack.
- API Gateway hardening covers headers, CORS, body limits, rate limiting, and consistent security error shape.
- Threat modeling and RBAC documentation show how the system is reviewed, not just coded.
- Multitenancy demonstrates practical tenant isolation without adding operational complexity such as database-per-tenant.
- Data lifecycle is treated operationally with retention docs, backups, archival exports, replay guidance, and explicit confirmation for destructive actions.
- Deployment readiness is represented with a simple Helm chart while keeping Compose as the primary local runtime.
- Event consumers are defensive so bad demo payloads do not crash the process.
- Rule simulation shows how to test risk-control changes safely before mutating production behavior.
- Admin operations APIs show how operators get health, catalog, DLQ, activity, and safe config visibility through the existing gateway boundary.
- The gateway keeps external routing stable while services retain internal base paths.
- Audit logging demonstrates compliance-style traceability without coupling business services to synchronous audit writes.
- The project includes demo scripts, release notes, architecture docs, troubleshooting, and production-readiness documentation because operational clarity is part of engineering quality.

## Known Limitations And Future Improvements

- This is a local Compose platform, not a production deployment.
- Kafka schemas are repository-local JSON Schemas but not enforced by a live schema registry.
- Correlation IDs are not a substitute for OpenTelemetry spans when deep latency attribution is needed.
- Consumer lag metrics are not implemented yet; current event-health views use retry, failure, duplicate, and DLQ metrics.
- Retention and archival scripts are local portfolio helpers, not regulated production archival automation.
- Helm is an optional readiness layer and does not install production-grade stateful infrastructure.
- Email delivery is mock-only.
- Managed secrets, TLS ingress, production database isolation, production-grade distributed abuse protection, and OpenTelemetry tracing are future work.
- Tenant-specific partitions, encryption keys, rate limits, schemas, or databases are future hardening options.
