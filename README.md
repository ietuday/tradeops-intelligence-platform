# TradeOps Intelligence Platform

TradeOps Intelligence Platform is an enterprise-style event-driven trading microservices platform built to demonstrate senior backend engineering concepts including Go microservices, API Gateway, JWT/RBAC, Kafka/Redpanda, MQTT ingestion, PostgreSQL, Redis, real-time WebSocket streaming, observability, reliability, audit trails, notifications, security hardening, Helm deployment readiness, and performance testing.

TradeOps is built as a portfolio and interview project: it models a realistic backend platform for simulated trading workflows while staying fully runnable on a local machine with Docker Compose.

Current release: `v2.9.0` Real-Time Dashboard Frontend Enhancement.

## Architecture Summary

The platform exposes a single API Gateway for client traffic and uses service-owned backend domains for identity, market data, orders, portfolio, strategy, risk, surveillance, notifications, and audit. Synchronous HTTP APIs handle commands and queries. Redpanda/Kafka connects asynchronous workflows such as order lifecycle events, portfolio updates, surveillance alerts, notification delivery, and audit logging.

Core infrastructure includes PostgreSQL, Redis, Mosquitto, Redpanda, Prometheus, Grafana, and Docker Compose.

v2.2.0 adds tenant-aware architecture using shared PostgreSQL tables with `tenant_id` columns, a standard JWT `tenantId` claim, `X-Tenant-ID` service propagation, tenant-aware events, audit records, and WebSocket filtering. See [tenant model](docs/multitenancy/tenant-model.md).

v2.3.0 adds local-demo OpenTelemetry tracing with Jaeger for the API Gateway, order, surveillance, notification, and audit flow while preserving `X-Correlation-ID` for logs, events, DLQ, and audit lookup. See [OpenTelemetry tracing](docs/tracing/opentelemetry.md).

v2.4.0 adds a lightweight SQL migration runner, `schema_migrations` tracking, checksum validation, and idempotent demo seed management. See [database migrations](docs/database/migrations.md).

v2.5.0 adds versioned JSON Schemas, an event envelope standard, a compatibility checklist, and read-only validation for Kafka/Redpanda and WebSocket event contracts. See [event schema governance](docs/events/schema-governance.md).

v2.6.0 adds tenant-aware, database-backed surveillance rule configuration APIs for thresholds, severity, and enable/disable state with environment defaults as fallback. See [surveillance rule configuration](docs/surveillance/rule-configuration.md).

v2.7.0 adds backend-only admin console APIs under `/api/admin` for health summaries, service/topic catalogs, DLQ guidance, audit/alert/notification/rule summaries, and safe platform config masking. See [admin console APIs](docs/admin/admin-console-apis.md).

v2.8.0 adds advanced risk analytics and stress testing APIs for scenario analysis, concentration risk, drawdown trends, volatility shocks, and explainable recommendations. See [advanced risk analytics](docs/risk/advanced-risk-analytics.md).

v2.9.0 adds a lightweight React real-time dashboard for admin summaries, WebSocket events, risk analytics demos, and observability links. See [real-time dashboard](docs/frontend/realtime-dashboard.md).

## Tech Stack

| Area | Technologies |
| --- | --- |
| API Gateway | Node.js, TypeScript, Express, Jest |
| Go services | Go, Chi, pgx, kafka-go, Prometheus client |
| Python services | FastAPI, SQLAlchemy, psycopg, confluent-kafka |
| Data | PostgreSQL, Redis |
| Messaging | Redpanda/Kafka, Mosquitto/MQTT |
| Event contracts | JSON Schema, versioned event catalog, compatibility rules |
| Observability | Prometheus, Grafana, Jaeger, OpenTelemetry, correlation IDs, health/readiness endpoints, metrics, alert rules, SLO dashboards |
| Security | JWT/RBAC, Helmet, CORS config, request size limits, rate limiting, security checklist |
| Multitenancy | Shared database tenant isolation, JWT `tenantId`, `X-Tenant-ID`, tenant-aware events |
| Real-time | API Gateway WebSocket streams for market, order, alert, notification, and audit events |
| Performance | Lightweight curl timing checks, optional k6 scenarios, capacity-planning docs |
| Runtime | Docker Compose, optional Helm/Kubernetes manifests, Makefile, Bash demo/smoke scripts |

## Services

| Service | Purpose | Port |
| --- | --- | --- |
| API Gateway | External HTTP entry point and service proxy | `8080` |
| Identity Service | Register/login, JWT, refresh tokens, RBAC | `8084` |
| Market Data Service | MQTT tick ingestion, validation, Kafka publishing | `8085` |
| Order Service | Simulated order lifecycle, idempotency, order events | `8086` |
| Portfolio Service | Consumes fills, updates holdings/cash, publishes snapshots | `8087` |
| Strategy Service | Strategy CRUD, backtesting, generated signals | `8088` |
| Risk Engine Service | Risk score, VaR, volatility, drawdown, stress testing, concentration analysis, recommendations | `8089` |
| Surveillance Service | Rule-based alerts from order/market/risk events | `8090` |
| Notification Service | Alert notifications, preferences, webhook/mock email delivery | `8091` |
| Audit Service | Searchable audit logs, summaries, exports, compliance event trail | `8092` |

## Feature Highlights

- Event-driven trading workflow across orders, portfolio, risk, surveillance, notifications, and audit.
- Go microservices for transactional domains, Python services for analytics-oriented domains, and a Node.js API Gateway.
- Real-time WebSocket streams for market ticks, order events, alerts, notifications, and audit events.
- Backend admin operations APIs for platform health, service/topic visibility, DLQ guidance, activity summaries, and safe runtime config views.
- Advanced risk analytics APIs for stress testing, scenario analysis, concentration risk, drawdown trends, and volatility shocks.
- Lightweight React dashboard for demoing admin operations, real-time streams, risk analytics, and observability links.
- Tenant-aware APIs, events, audit logs, and WebSocket streams using `default-tenant` for local demos.
- JWT/RBAC, idempotent order creation, retries/DLQ guidance, audit exports, and correlation IDs.
- Prometheus metrics, Grafana dashboards, SLO docs, observability runbooks, and performance testing scripts.
- Jaeger/OpenTelemetry tracing for the primary order-to-alert-to-notification-to-audit path.
- Docker Compose local runtime with optional Helm/Kubernetes deployment-readiness artifacts.

## Portfolio Pack

| Resource | Link |
| --- | --- |
| Project overview | [docs/portfolio/project-overview.md](docs/portfolio/project-overview.md) |
| Interview pitch | [docs/portfolio/interview-pitch.md](docs/portfolio/interview-pitch.md) |
| Demo walkthrough | [docs/portfolio/demo-walkthrough.md](docs/portfolio/demo-walkthrough.md) |
| Resume bullets | [docs/portfolio/resume-bullets.md](docs/portfolio/resume-bullets.md) |
| LinkedIn/GitHub summary | [docs/portfolio/linkedin-github-summary.md](docs/portfolio/linkedin-github-summary.md) |
| Interview Q&A | [docs/interview/tradeops-interview-qa.md](docs/interview/tradeops-interview-qa.md) |
| Architecture summary | [docs/portfolio/architecture-summary.md](docs/portfolio/architecture-summary.md) |
| Final validation checklist | [docs/portfolio/final-validation-checklist.md](docs/portfolio/final-validation-checklist.md) |

## Quick Start

Create a local Docker environment file:

```bash
cp infrastructure/docker/.env.example infrastructure/docker/.env
```

Start the platform:

```bash
make dev-up
```

Check service status:

```bash
make ps
make smoke
```

Run database migrations and demo seeds against local PostgreSQL:

```bash
./scripts/db-migrate.sh
./scripts/db-seed.sh
```

Validate event schemas and mapped sample payloads:

```bash
./scripts/validate-event-schemas.sh
```

Validate Compose config in CI-style mode without relying on a local `.env`:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml config
```

Stop the platform:

```bash
make dev-down
```

## Demo Commands

Run the full end-to-end demo:

```bash
./scripts/demo-e2e-tradeops.sh
```

Run focused demos:

```bash
./scripts/demo-surveillance.sh
./scripts/demo-rule-config.sh
./scripts/demo-rule-simulation.sh
./scripts/demo-notifications.sh
./scripts/demo-audit.sh
./scripts/demo-reliability.sh
./scripts/demo-observability.sh
./scripts/demo-correlation-tracing.sh
./scripts/demo-otel-tracing.sh
./scripts/demo-db-migrations.sh
./scripts/demo-risk-analytics.sh
./scripts/demo-dashboard.sh
```

Validate scripts without running the platform:

```bash
bash -n scripts/smoke-test.sh
bash -n scripts/run-load-tests.sh
bash -n scripts/perf-smoke.sh
bash -n scripts/demo-surveillance.sh
bash -n scripts/demo-rule-config.sh
bash -n scripts/demo-rule-simulation.sh
bash -n scripts/demo-notifications.sh
bash -n scripts/demo-audit.sh
bash -n scripts/demo-e2e-tradeops.sh
bash -n scripts/demo-reliability.sh
bash -n scripts/demo-observability.sh
bash -n scripts/demo-correlation-tracing.sh
bash -n scripts/demo-otel-tracing.sh
bash -n scripts/validate-event-schemas.sh
bash -n scripts/db-migrate.sh
bash -n scripts/db-seed.sh
bash -n scripts/demo-db-migrations.sh
bash -n scripts/demo-websocket-streams.sh
bash -n scripts/demo-risk-analytics.sh
bash -n scripts/demo-dashboard.sh
bash -n scripts/db-backup.sh
bash -n scripts/db-restore.sh
bash -n scripts/archive-old-data.sh
bash -n scripts/replay-sample-events.sh
bash -n scripts/replay-dlq-events.sh
bash -n scripts/validate-helm.sh
```

## Local URLs

| Component | URL |
| --- | --- |
| API Gateway | http://localhost:8080 |
| Redpanda Console | http://localhost:8081 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |
| Jaeger | http://localhost:16686 |
| Angular Shell Placeholder | http://localhost:4200 |
| React Real-Time Dashboard | http://localhost:4300 |

## Observability & SLOs

TradeOps includes Prometheus scraping, Grafana dashboard provisioning, local alert rules, and SLO-oriented documentation for demo and interview walkthroughs.

| Asset | Location |
| --- | --- |
| Metrics catalog | [docs/observability/metrics-catalog.md](docs/observability/metrics-catalog.md) |
| Grafana dashboard guide | [docs/observability/grafana-dashboards.md](docs/observability/grafana-dashboards.md) |
| Prometheus alert guide | [docs/observability/prometheus-alerts.md](docs/observability/prometheus-alerts.md) |
| SLO guide | [docs/observability/slo-guide.md](docs/observability/slo-guide.md) |
| Observability runbook | [docs/observability/runbook.md](docs/observability/runbook.md) |
| OpenTelemetry tracing | [docs/tracing/opentelemetry.md](docs/tracing/opentelemetry.md) |
| OpenTelemetry runbook | [docs/tracing/otel-runbook.md](docs/tracing/otel-runbook.md) |
| Alert rules | [infrastructure/docker/prometheus/rules/tradeops-alerts.yml](infrastructure/docker/prometheus/rules/tradeops-alerts.yml) |
| Grafana dashboards | [infrastructure/docker/grafana/dashboards](infrastructure/docker/grafana/dashboards) |

Run the read-only observability demo:

```bash
./scripts/demo-observability.sh
```

## Database Migrations & Seeds

TradeOps includes a simple SQL+Bash migration workflow for local demos and CI-friendly validation:

```bash
./scripts/db-migrate.sh
./scripts/db-seed.sh
./scripts/demo-db-migrations.sh
```

The runner creates `schema_migrations`, applies sorted SQL migrations once, validates checksums, and keeps seeds idempotent with `ON CONFLICT`. See [database migrations](docs/database/migrations.md), [seed management](docs/database/seed-management.md), and the [database runbook](docs/database/db-runbook.md).

## Event Schema Governance

Event contracts live under `schemas/events/` as versioned JSON Schema files. v2.5.0 keeps validation lightweight and repository-local: schemas document current flat payloads while allowing optional envelope metadata such as `eventVersion`, `tenantId`, `correlationId`, and `traceparent`.

```bash
./scripts/validate-event-schemas.sh
```

Start with the [event catalog](docs/events/event-catalog.md), [event envelope](docs/events/event-envelope.md), [compatibility rules](docs/events/compatibility-rules.md), and [event contract checklist](docs/events/event-contract-checklist.md).

## Surveillance Rule Configuration

Surveillance rule thresholds and enable/disable state are configurable per tenant through `/api/surveillance/rules`. Existing env vars remain fallback defaults, so local demos continue to work if the config table is empty.

```bash
TOKEN=<jwt> ./scripts/demo-rule-config.sh
TOKEN=<jwt> ./scripts/demo-rule-config.sh --apply
```

See [rule configuration](docs/surveillance/rule-configuration.md).

## Surveillance Rule Simulation

Dry-run proposed surveillance rule configs without updating `surveillance_rule_configs`, refreshing live caches, creating alerts, or triggering notifications:

```bash
TOKEN=<jwt> ./scripts/demo-rule-simulation.sh
```

Example API call:

```bash
curl -X POST "http://localhost:8080/api/surveillance/rules/LargeOrderRule/simulate" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"tenantId":"default-tenant","lookbackMinutes":60,"dryRun":true,"config":{"thresholdNumeric":200000}}'
```

See [rule simulation](docs/surveillance/rule-simulation.md).

## Real-Time WebSocket Streaming

The API Gateway exposes lightweight WebSocket streams for live platform events:

```text
/ws
/ws/market
/ws/orders
/ws/alerts
/ws/notifications
/ws/audit
```

Try the safe demo:

```bash
TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --orders
TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --alerts --publish-sample
```

See [WebSocket streaming](docs/realtime/websocket-streaming.md) and the [real-time runbook](docs/realtime/realtime-runbook.md).

## Multitenancy

TradeOps uses shared database multitenancy with additive `tenant_id` columns and application-level tenant filtering. JWTs include `tenantId`, API Gateway forwards `X-Tenant-ID`, tenant-owned Kafka events include `tenantId`, and WebSocket streams filter tenant events by connection tenant.

Local demos default to:

```bash
TENANT_ID=default-tenant
```

Start here: [tenant model](docs/multitenancy/tenant-model.md), [tenant isolation](docs/multitenancy/tenant-isolation.md), [migration guide](docs/multitenancy/migration-guide.md), and [runbook](docs/multitenancy/runbook.md).

## Performance Testing & Capacity Planning

TradeOps includes safe local performance checks and optional k6 scenarios. These are demo baselines for one machine, not production capacity numbers.

```bash
./scripts/perf-smoke.sh
./scripts/run-load-tests.sh --gateway
BASE_URL=http://localhost:8080 TOKEN=<jwt> ./scripts/run-load-tests.sh --surveillance
```

If k6 is not installed, the load-test runner skips gracefully and prints install guidance. During tests, watch API Gateway p95 latency, 5xx rate, upstream errors/timeouts, event failures, retries, DLQ metrics, and notification delivery failures.

Performance references: [load testing](docs/performance/load-testing.md), [benchmark plan](docs/performance/benchmark-plan.md), [capacity planning](docs/performance/capacity-planning.md), [performance runbook](docs/performance/performance-runbook.md), and [results template](docs/performance/performance-results-template.md).

## Distributed Tracing & Correlation Visibility

TradeOps uses lightweight correlation IDs instead of a heavy tracing stack. The standard HTTP header is `X-Correlation-ID`, Kafka/JSON events use `correlationId`, logs use `correlationId`, and audit logs persist `audit_logs.correlation_id`.

Run the safe tracing demo:

```bash
./scripts/demo-correlation-tracing.sh
CORRELATION_ID=demo-correlation-123 ./scripts/demo-correlation-tracing.sh --publish-sample
```

Grep logs by correlation ID:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway order-service surveillance-service notification-service audit-service | grep demo-correlation-123
```

Query audit logs by correlation ID:

```bash
curl "http://localhost:8080/api/audit/logs?correlationId=demo-correlation-123" \
  -H "Authorization: Bearer ${TOKEN}"
```

See [correlation standard](docs/tracing/correlation-standard.md), [structured logging](docs/tracing/structured-logging.md), and [tracing runbook](docs/tracing/tracing-runbook.md). OpenTelemetry tracing is a future enhancement.

## Security Hardening

TradeOps keeps security practical and local-demo friendly: JWT/RBAC remains enforced by backend services where implemented, the API Gateway uses Helmet security headers, configurable CORS, request body limits, in-memory rate limiting, proxy timeout handling, and consistent error responses with `correlationId`.

Key gateway settings:

```bash
CORS_ORIGIN=http://localhost:4200,http://localhost:4300
REQUEST_BODY_LIMIT=1mb
RATE_LIMIT_WINDOW_MS=60000
RATE_LIMIT_MAX_REQUESTS=300
```

Run the read-only security check:

```bash
./scripts/security-check.sh
make security-check
```

Security references: [threat model](docs/security/threat-model.md), [RBAC matrix](docs/security/rbac-matrix.md), [API security](docs/security/api-security.md), [secrets management](docs/security/secrets-management.md), and [security checklist](docs/security/security-checklist.md).

## Data Retention, Backup & Replay

TradeOps includes local data lifecycle guidance and safe helper scripts for backups, archival exports, sample replay, and DLQ replay. Destructive operations are not automatic.

| Task | Command Or Guide |
| --- | --- |
| Retention policy | [docs/data-lifecycle/retention-policy.md](docs/data-lifecycle/retention-policy.md) |
| Backup PostgreSQL | `./scripts/db-backup.sh` |
| Restore PostgreSQL | `./scripts/db-restore.sh backups/file.sql --confirm` |
| Archive old data dry-run/export | `./scripts/archive-old-data.sh` |
| Replay sample events | `./scripts/replay-sample-events.sh --all` |
| DLQ replay guidance | [docs/data-lifecycle/dlq-replay.md](docs/data-lifecycle/dlq-replay.md) |
| Data lifecycle runbook | [docs/data-lifecycle/runbook.md](docs/data-lifecycle/runbook.md) |

Safety note: restore requires `--confirm`, archive deletion requires `--delete-confirm`, and DLQ replay remains manual/conservative by design.

## Deployment Readiness

Docker Compose remains the primary local demo runtime. The optional Helm chart is provided for Kubernetes deployment-readiness discussion and local rendering checks.

| Asset | Location |
| --- | --- |
| Helm chart | [infrastructure/helm/tradeops-platform](infrastructure/helm/tradeops-platform) |
| Helm chart README | [infrastructure/helm/tradeops-platform/README.md](infrastructure/helm/tradeops-platform/README.md) |
| Kubernetes/Helm guide | [docs/deployment/kubernetes-helm.md](docs/deployment/kubernetes-helm.md) |
| Deployment readiness checklist | [docs/deployment/deployment-readiness.md](docs/deployment/deployment-readiness.md) |

Validate the chart:

```bash
./scripts/validate-helm.sh
make validate-helm
```

## Screenshots

No screenshots are committed by default. Use the [portfolio screenshots guide](docs/portfolio/screenshots-guide.md) for suggested captures such as Docker Compose services, Prometheus targets, Grafana dashboards, Redpanda topics, API responses, demo output, Helm template output, and GitHub Actions status.

## Documentation

- [Documentation index](docs/index.md)
- [Architecture overview](docs/architecture/overview.md)
- [Event-flow reference](docs/architecture/event-flow.md)
- [Service dependency matrix](docs/architecture/service-dependency-matrix.md)
- [API summary](docs/api/api-summary.md)
- [Audit trail](docs/audit/audit-trail.md)
- [Surveillance rule simulation](docs/surveillance/rule-simulation.md)
- [CI/CD quality gates](docs/ci-cd/quality-gates.md)
- [Observability metrics catalog](docs/observability/metrics-catalog.md)
- [Grafana dashboard guide](docs/observability/grafana-dashboards.md)
- [Prometheus alert guide](docs/observability/prometheus-alerts.md)
- [SLO guide](docs/observability/slo-guide.md)
- [Observability runbook](docs/observability/runbook.md)
- [Correlation ID standard](docs/tracing/correlation-standard.md)
- [Structured logging guidance](docs/tracing/structured-logging.md)
- [Tracing runbook](docs/tracing/tracing-runbook.md)
- [WebSocket streaming](docs/realtime/websocket-streaming.md)
- [Real-time runbook](docs/realtime/realtime-runbook.md)
- [Threat model](docs/security/threat-model.md)
- [RBAC matrix](docs/security/rbac-matrix.md)
- [API security guide](docs/security/api-security.md)
- [Secrets management guide](docs/security/secrets-management.md)
- [Security checklist](docs/security/security-checklist.md)
- [Performance load testing](docs/performance/load-testing.md)
- [Benchmark plan](docs/performance/benchmark-plan.md)
- [Capacity planning](docs/performance/capacity-planning.md)
- [Performance runbook](docs/performance/performance-runbook.md)
- [Performance results template](docs/performance/performance-results-template.md)
- [Data retention policy](docs/data-lifecycle/retention-policy.md)
- [Archival strategy](docs/data-lifecycle/archival-strategy.md)
- [Backup and restore guide](docs/data-lifecycle/backup-restore.md)
- [Event replay guide](docs/data-lifecycle/event-replay.md)
- [DLQ replay guide](docs/data-lifecycle/dlq-replay.md)
- [Data lifecycle runbook](docs/data-lifecycle/runbook.md)
- [Kubernetes/Helm guide](docs/deployment/kubernetes-helm.md)
- [Deployment readiness checklist](docs/deployment/deployment-readiness.md)
- [Reliability patterns](docs/reliability/resilience-patterns.md)
- [Dead-letter topics](docs/reliability/dead-letter-topics.md)
- [Graceful shutdown](docs/reliability/graceful-shutdown.md)
- [Production-readiness checklist](docs/production-readiness/checklist.md)
- [Troubleshooting guide](docs/troubleshooting.md)
- [Repository cleanup guide](docs/release/repository-cleanup.md)
- [Screenshot guide](docs/screenshots/README.md)
- [Portfolio screenshots guide](docs/portfolio/screenshots-guide.md)
- [Portfolio project overview](docs/portfolio/project-overview.md)
- [Portfolio interview pitch](docs/portfolio/interview-pitch.md)
- [Portfolio demo walkthrough](docs/portfolio/demo-walkthrough.md)
- [Portfolio architecture summary](docs/portfolio/architecture-summary.md)
- [Portfolio final roadmap](docs/portfolio/future-roadmap.md)
- [Final validation checklist](docs/portfolio/final-validation-checklist.md)
- [Interview walkthrough](docs/interview/project-walkthrough.md)
- [Interview Q&A](docs/interview/tradeops-interview-qa.md)
- [Resume bullets](docs/interview/resume-bullets.md)
- [Portfolio resume bullets](docs/portfolio/resume-bullets.md)
- [LinkedIn/GitHub project summary](docs/interview/project-summary-for-linkedin.md)
- [Portfolio LinkedIn/GitHub summary](docs/portfolio/linkedin-github-summary.md)

## Release Notes

- [v2.8.0 Advanced Risk Analytics & Stress Testing](docs/release-notes/v2.8.0.md)
- [v2.6.0 Rule Configuration Management](docs/release-notes/v2.6.0.md)
- [v2.1.0 Real-Time WebSocket Streaming Layer](docs/release-notes/v2.1.0.md)
- [v2.0.0 Final Portfolio Release](docs/release-notes/v2.0.0.md)
- [v1.9.0 Performance Testing, Load Testing & Capacity Planning](docs/release-notes/v1.9.0.md)
- [v1.8.0 Security Hardening & Threat Modeling](docs/release-notes/v1.8.0.md)
- [v1.7.0 Distributed Tracing & Correlation Visibility](docs/release-notes/v1.7.0.md)
- [v1.6.0 Deployment Readiness: Kubernetes / Helm Optional Layer](docs/release-notes/v1.6.0.md)
- [v1.5.0 Data Retention, Archival & Event Replay](docs/release-notes/v1.5.0.md)
- [v1.4.0 Advanced Observability & SLO Dashboards](docs/release-notes/v1.4.0.md)
- [v1.3.0 Audit Trail & Compliance Reporting](docs/release-notes/v1.3.0.md)
- [v1.2.0 Reliability, Resilience & Failure Handling](docs/release-notes/v1.2.0.md)
- [v1.1.0 CI/CD, Security Scanning & Quality Gates](docs/release-notes/v1.1.0.md)
- [v1.0.1 GitHub Release & Portfolio Polish](docs/release-notes/v1.0.1.md)
- [v1.0.0 Production Readiness & Platform Hardening](docs/release-notes/v1.0.0.md)
- [Earlier release notes](docs/release-notes/)

## CI/CD & Quality Gates

GitHub Actions workflows live under `.github/workflows/`:

| Workflow | Purpose |
| --- | --- |
| `ci.yml` | Node, Go, Python, script, and Docker Compose validation. |
| `security.yml` | Secret scanning, vulnerability checks, audits, and static security checks. |
| `docker.yml` | Builds all service Docker images with local CI tags. |
| `docs.yml` | Validates required docs and runs non-blocking Markdown linting. |

Common local commands:

```bash
make help
make validate-scripts
make security-check
make perf-smoke
make load-test-gateway
make compose-config
make test-node
make test-go
make test-python
make docker-build
```

See [CI/CD quality gates](docs/ci-cd/quality-gates.md) for workflow details, security scanning notes, and local validation guidance.

## Production-Readiness Note

TradeOps demonstrates production-oriented backend practices: service boundaries, JWT/RBAC, API Gateway hardening, threat modeling, idempotency, event-driven integration, audit trails, correlation IDs, health/readiness checks, metrics, Prometheus alerts, SLO dashboards, performance baselines, capacity-planning guidance, data retention guidance, backup/replay scripts, optional Helm deployment manifests, smoke tests, demo scripts, release notes, troubleshooting docs, and Grafana dashboards.

It is still a local portfolio platform, not a real production deployment. See the [production-readiness checklist](docs/production-readiness/checklist.md) for honest gaps and future hardening work.

## Known Limitations

- Docker Compose is used for local orchestration only.
- Event schemas are documented by examples, not enforced by a schema registry.
- Surveillance rule simulation uses deterministic demo/historical-style events rather than a production historical event store.
- Audit export is API-returned JSON/CSV, not durable file generation.
- Data lifecycle scripts are local operational helpers, not regulated production retention automation.
- Helm manifests are deployment-readiness artifacts, not a fully managed production cluster setup.
- Correlation IDs are lightweight tracing aids, not full distributed tracing spans.
- API Gateway rate limiting is in-memory and intended for local/demo use, not distributed production abuse protection.
- Local performance tests are not production capacity benchmarks.
- Notification email delivery is mock/log-only.
- Frontend apps are placeholders/foundations, not complete trading UIs.
- No cloud deployment, TLS ingress, OAuth/OIDC provider, mTLS, WAF, or managed secret store is included yet.

## Future Roadmap

- Add CI/CD pipeline documentation and automated release checks.
- Add OpenAPI specs for gateway routes.
- Add OpenTelemetry tracing when span-level visibility is worth the added infrastructure.
- Add production identity provider integration, TLS ingress, WAF/rate-limit integration, and managed secrets.
- Add automated performance regression gates after stable baselines exist.
- Add schema validation or schema registry for Kafka events.
- Add richer portfolio screenshots.
- Add production-grade Kubernetes hardening after the optional Helm layer is validated against a real cluster.

## Validation

Recommended release validation:

```bash
(cd services/surveillance-service && go test ./...)
(cd services/notification-service && go test ./...)
(cd services/audit-service && go test ./...)
(cd services/api-gateway && npm test -- --runInBand)
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml config
bash -n scripts/run-load-tests.sh
bash -n scripts/perf-smoke.sh
bash -n scripts/smoke-test.sh
bash -n scripts/demo-surveillance.sh
bash -n scripts/demo-rule-simulation.sh
bash -n scripts/demo-notifications.sh
bash -n scripts/demo-audit.sh
bash -n scripts/demo-e2e-tradeops.sh
bash -n scripts/demo-reliability.sh
bash -n scripts/demo-observability.sh
bash -n scripts/demo-correlation-tracing.sh
bash -n scripts/demo-websocket-streams.sh
bash -n scripts/security-check.sh
bash -n scripts/db-backup.sh
bash -n scripts/db-restore.sh
bash -n scripts/archive-old-data.sh
bash -n scripts/replay-sample-events.sh
bash -n scripts/replay-dlq-events.sh
bash -n scripts/validate-helm.sh
./scripts/validate-helm.sh
./scripts/security-check.sh
```
