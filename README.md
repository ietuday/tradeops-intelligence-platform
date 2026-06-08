# TradeOps Intelligence Platform

Enterprise-style local trading intelligence platform with microservices, event-driven workflows, JWT/RBAC, audit trails, observability, demo scripts, and production-readiness documentation.

TradeOps is built as a portfolio and interview project: it models a realistic backend platform for simulated trading workflows while staying fully runnable on a local machine with Docker Compose.

Latest release: `v1.3.0` Audit Trail & Compliance Reporting.

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
| Observability | Prometheus, Grafana, health/readiness endpoints, metrics |
| Runtime | Docker Compose, Makefile, Bash demo/smoke scripts |

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
```

Validate scripts without running the platform:

```bash
bash -n scripts/smoke-test.sh
bash -n scripts/demo-surveillance.sh
bash -n scripts/demo-notifications.sh
bash -n scripts/demo-audit.sh
bash -n scripts/demo-e2e-tradeops.sh
bash -n scripts/demo-reliability.sh
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

## Documentation

- [Architecture overview](docs/architecture/overview.md)
- [Event-flow reference](docs/architecture/event-flow.md)
- [Service dependency matrix](docs/architecture/service-dependency-matrix.md)
- [API summary](docs/api/api-summary.md)
- [Audit trail](docs/audit/audit-trail.md)
- [CI/CD quality gates](docs/ci-cd/quality-gates.md)
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

TradeOps demonstrates production-oriented backend practices: service boundaries, JWT/RBAC, idempotency, event-driven integration, audit trails, health/readiness checks, metrics, smoke tests, demo scripts, release notes, troubleshooting docs, and Grafana dashboards.

It is still a local portfolio platform, not a real production deployment. See the [production-readiness checklist](docs/production-readiness/checklist.md) for honest gaps and future hardening work.

## Known Limitations

- Docker Compose is used for local orchestration only.
- Event schemas are documented by examples, not enforced by a schema registry.
- Audit export is API-returned JSON/CSV, not durable file generation.
- Notification email delivery is mock/log-only.
- Frontend apps are placeholders/foundations, not complete trading UIs.
- No Kubernetes, Helm, cloud deployment, TLS ingress, or managed secret store is included yet.

## Future Roadmap

- Add CI/CD pipeline documentation and automated release checks.
- Add OpenAPI specs for gateway routes.
- Add distributed tracing and alert rules.
- Add schema validation or schema registry for Kafka events.
- Add richer Grafana dashboards and portfolio screenshots.
- Add Kubernetes/Helm only after the local platform is stable enough to justify it.

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
```
