# Resume Bullets

## 3 Short Bullets

- Built an event-driven trading microservices platform using Go, Python FastAPI, Node.js, PostgreSQL, Redis, Redpanda/Kafka, MQTT, Prometheus, Grafana, and Docker Compose.
- Implemented order lifecycle, portfolio updates, risk scoring, surveillance alerts, notifications, audit trails, idempotency, retries/DLQ guidance, and correlation tracing.
- Added production-readiness layers including CI/CD quality gates, security hardening, SLO dashboards, Helm templates, backup/replay scripts, and performance testing.

## 5 Detailed Bullets

- Designed a multi-service trading operations backend with API Gateway routing, JWT/RBAC authentication, service-owned domains, PostgreSQL persistence, Redis identity state, MQTT ingestion, and Kafka-compatible event flows.
- Implemented event-driven workflows for orders, portfolio updates, risk scoring, surveillance alert lifecycle, notification delivery attempts, and searchable audit/compliance logs.
- Added reliability patterns including idempotent order creation, defensive event parsing, retry/backoff handling, DLQ documentation, sample event replay, backup/restore scripts, and smoke-test automation.
- Built observability assets with Prometheus metrics, Grafana dashboards, SLO guidance, alert rules, correlation IDs across HTTP/events/logs, and operational runbooks.
- Prepared production-readiness documentation covering security threat modeling, RBAC, secrets management, optional Helm deployment templates, performance testing, capacity planning, and release validation.

## Go Backend Focused

- Built Go microservices for identity, market data, orders, portfolio, surveillance, notifications, and audit with HTTP APIs, PostgreSQL migrations, Kafka integration, and Prometheus metrics.
- Implemented domain logic for order validation/idempotency, portfolio updates, surveillance rules, notification delivery attempts, and audit ingestion.
- Added focused unit tests, health/readiness endpoints, graceful operational patterns, and Dockerized service runtimes.

## Microservices/Platform Focused

- Designed a gateway-led microservices architecture with synchronous APIs for commands/queries and Kafka events for asynchronous business workflows.
- Standardized health, readiness, metrics, smoke tests, demo scripts, release notes, troubleshooting docs, and service dependency documentation across the platform.

## Cloud/Kubernetes Focused

- Added optional Helm chart readiness with Deployments, Services, ConfigMap/Secret separation, probes, resource limits, and ingress placeholders.
- Documented production gaps around managed PostgreSQL/Kafka/Redis, TLS ingress, network policy, external secrets, autoscaling, and rollout strategies.

## Observability/Reliability Focused

- Built observability and reliability layers with Prometheus/Grafana dashboards, SLO docs, alert rules, correlation IDs, retry/DLQ guidance, replay scripts, and performance runbooks.
