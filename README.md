# TradeOps Intelligence Platform

Enterprise-style local trading intelligence platform with microservices, event-driven workflows, JWT/RBAC, audit trails, observability, demo scripts, and production-readiness documentation.

TradeOps is built as a portfolio and interview project: it models a realistic backend platform for simulated trading workflows while staying fully runnable on a local machine with Docker Compose.

Latest release: `v1.7.0` Distributed Tracing & Correlation Visibility.

## Architecture Summary

The platform exposes a single API Gateway for client traffic and uses service-owned backend domains for identity, market data, orders, portfolio, strategy, risk, surveillance, notifications, and audit. Synchronous HTTP APIs handle commands and queries. Redpanda/Kafka connects asynchronous workflows such as order lifecycle events, portfolio updates, surveillance alerts, notification delivery, and audit logging.

Core infrastructure includes PostgreSQL, Redis, Mosquitto, Redpanda, Prometheus, Grafana, and Docker Compose.

## Tech Stack

| Area | Technologies |
| --- | --- |
| API Gateway | Node.js, TypeScript, Express, Jest |
| Go services | Go, Chi, pgx, kafka-go, Prometheus client |
| Python services | FastAPI, SQLAlchemy, psycopg, confluent-kafka |
| Data | PostgreSQL, Redis |
| Messaging | Redpanda/Kafka, Mosquitto/MQTT |
| Observability | Prometheus, Grafana, correlation IDs, health/readiness endpoints, metrics, alert rules, SLO dashboards |
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
| Risk Engine Service | Risk score, VaR, volatility, drawdown, recommendations | `8089` |
| Surveillance Service | Rule-based alerts from order/market/risk events | `8090` |
| Notification Service | Alert notifications, preferences, webhook/mock email delivery | `8091` |
| Audit Service | Searchable audit logs, summaries, exports, compliance event trail | `8092` |

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
./scripts/demo-notifications.sh
./scripts/demo-audit.sh
./scripts/demo-reliability.sh
./scripts/demo-observability.sh
./scripts/demo-correlation-tracing.sh
```

Validate scripts without running the platform:

```bash
bash -n scripts/smoke-test.sh
bash -n scripts/demo-surveillance.sh
bash -n scripts/demo-notifications.sh
bash -n scripts/demo-audit.sh
bash -n scripts/demo-e2e-tradeops.sh
bash -n scripts/demo-reliability.sh
bash -n scripts/demo-observability.sh
bash -n scripts/demo-correlation-tracing.sh
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
| Angular Shell Placeholder | http://localhost:4200 |
| React Dashboard Placeholder | http://localhost:4300 |

## Observability & SLOs

TradeOps includes Prometheus scraping, Grafana dashboard provisioning, local alert rules, and SLO-oriented documentation for demo and interview walkthroughs.

| Asset | Location |
| --- | --- |
| Metrics catalog | [docs/observability/metrics-catalog.md](docs/observability/metrics-catalog.md) |
| Grafana dashboard guide | [docs/observability/grafana-dashboards.md](docs/observability/grafana-dashboards.md) |
| Prometheus alert guide | [docs/observability/prometheus-alerts.md](docs/observability/prometheus-alerts.md) |
| SLO guide | [docs/observability/slo-guide.md](docs/observability/slo-guide.md) |
| Observability runbook | [docs/observability/runbook.md](docs/observability/runbook.md) |
| Alert rules | [infrastructure/docker/prometheus/rules/tradeops-alerts.yml](infrastructure/docker/prometheus/rules/tradeops-alerts.yml) |
| Grafana dashboards | [infrastructure/docker/grafana/dashboards](infrastructure/docker/grafana/dashboards) |

Run the read-only observability demo:

```bash
./scripts/demo-observability.sh
```

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

## Documentation

- [Architecture overview](docs/architecture/overview.md)
- [Event-flow reference](docs/architecture/event-flow.md)
- [Service dependency matrix](docs/architecture/service-dependency-matrix.md)
- [API summary](docs/api/api-summary.md)
- [Audit trail](docs/audit/audit-trail.md)
- [CI/CD quality gates](docs/ci-cd/quality-gates.md)
- [Observability metrics catalog](docs/observability/metrics-catalog.md)
- [Grafana dashboard guide](docs/observability/grafana-dashboards.md)
- [Prometheus alert guide](docs/observability/prometheus-alerts.md)
- [SLO guide](docs/observability/slo-guide.md)
- [Observability runbook](docs/observability/runbook.md)
- [Correlation ID standard](docs/tracing/correlation-standard.md)
- [Structured logging guidance](docs/tracing/structured-logging.md)
- [Tracing runbook](docs/tracing/tracing-runbook.md)
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
- [Interview walkthrough](docs/interview/project-walkthrough.md)
- [Resume bullets](docs/interview/resume-bullets.md)
- [LinkedIn/GitHub project summary](docs/interview/project-summary-for-linkedin.md)

## Release Notes

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
make compose-config
make test-node
make test-go
make test-python
make docker-build
```

See [CI/CD quality gates](docs/ci-cd/quality-gates.md) for workflow details, security scanning notes, and local validation guidance.

## Production-Readiness Note

TradeOps demonstrates production-oriented backend practices: service boundaries, JWT/RBAC, idempotency, event-driven integration, audit trails, correlation IDs, health/readiness checks, metrics, Prometheus alerts, SLO dashboards, data retention guidance, backup/replay scripts, optional Helm deployment manifests, smoke tests, demo scripts, release notes, troubleshooting docs, and Grafana dashboards.

It is still a local portfolio platform, not a real production deployment. See the [production-readiness checklist](docs/production-readiness/checklist.md) for honest gaps and future hardening work.

## Known Limitations

- Docker Compose is used for local orchestration only.
- Event schemas are documented by examples, not enforced by a schema registry.
- Audit export is API-returned JSON/CSV, not durable file generation.
- Data lifecycle scripts are local operational helpers, not regulated production retention automation.
- Helm manifests are deployment-readiness artifacts, not a fully managed production cluster setup.
- Correlation IDs are lightweight tracing aids, not full distributed tracing spans.
- Notification email delivery is mock/log-only.
- Frontend apps are placeholders/foundations, not complete trading UIs.
- No Kubernetes, Helm, cloud deployment, TLS ingress, or managed secret store is included yet.

## Future Roadmap

- Add CI/CD pipeline documentation and automated release checks.
- Add OpenAPI specs for gateway routes.
- Add OpenTelemetry tracing when span-level visibility is worth the added infrastructure.
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
docker compose -f infrastructure/docker/docker-compose.yml config
bash -n scripts/smoke-test.sh
bash -n scripts/demo-surveillance.sh
bash -n scripts/demo-notifications.sh
bash -n scripts/demo-audit.sh
bash -n scripts/demo-e2e-tradeops.sh
bash -n scripts/demo-reliability.sh
bash -n scripts/demo-observability.sh
bash -n scripts/demo-correlation-tracing.sh
bash -n scripts/db-backup.sh
bash -n scripts/db-restore.sh
bash -n scripts/archive-old-data.sh
bash -n scripts/replay-sample-events.sh
bash -n scripts/replay-dlq-events.sh
bash -n scripts/validate-helm.sh
./scripts/validate-helm.sh
```
