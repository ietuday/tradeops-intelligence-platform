# Production Readiness Checklist

TradeOps is currently a local portfolio platform. It demonstrates production-oriented patterns, but it is not yet a real production deployment. Use this checklist to explain what is ready, what is partially ready, and what still needs hardening.

## Configuration

- [x] Services load configuration from environment variables.
- [x] Docker Compose has local defaults for non-secret values.
- [x] Required secrets are documented in `infrastructure/docker/.env.example`.
- [ ] Add environment-specific config validation for staging/production.
- [ ] Split local, staging, and production configuration profiles.

## Secrets

- [x] Local secrets are kept out of git through `.env`.
- [x] JWT secrets are shared consistently across services in local Compose.
- [ ] Move real deployments to a secret manager.
- [ ] Add secret rotation procedures.
- [ ] Avoid default local secrets outside development.

## Database Migrations

- [x] Services include startup migrations.
- [x] Tables are scoped by service domain.
- [ ] Add migration version tracking and rollback procedures.
- [ ] Run migrations as a controlled deployment step in production.
- [ ] Consider separate databases or schemas per service.

## Authentication And RBAC

- [x] Identity service issues JWT access tokens.
- [x] Services validate JWTs locally.
- [x] RBAC middleware exists for protected service APIs.
- [ ] Formalize role matrix per endpoint.
- [ ] Add token revocation strategy for access tokens if required.
- [ ] Add SSO/OIDC integration for real enterprise deployment.

## API Gateway

- [x] Gateway centralizes client HTTP access.
- [x] Gateway propagates authorization and correlation headers.
- [x] Gateway exposes health, readiness, and metrics.
- [ ] Add explicit request timeout, retry, and circuit-breaker policies per upstream.
- [ ] Add production-grade rate limiting and abuse protection.
- [ ] Generate OpenAPI specs for gateway routes.

## Messaging

- [x] Redpanda/Kafka topics connect order, portfolio, risk, surveillance, and notification flows.
- [x] Consumers handle malformed payloads defensively in key services.
- [x] Demo payloads exist for surveillance and notification events.
- [x] Sample replay and DLQ replay guidance are documented.
- [ ] Add schema validation or schema registry contracts.
- [x] Define dead-letter topics and conservative replay procedures.
- [ ] Add consumer lag monitoring.

## Data Lifecycle

- [x] Local retention policy is documented for market, order, portfolio, risk, surveillance, notification, audit, strategy/backtest, and DLQ data.
- [x] PostgreSQL backup and restore scripts are included.
- [x] Old-data archival export script is dry-run/export by default.
- [x] Destructive restore/delete actions require explicit confirmation flags.
- [x] Sample event replay and DLQ replay helpers are included.
- [ ] Add production-approved retention schedules and legal/compliance review.
- [ ] Add managed archive storage or warehouse integration for real deployments.

## Observability

- [x] Services expose Prometheus metrics.
- [x] Docker Compose includes Prometheus and Grafana.
- [x] Grafana dashboard exports cover platform overview, API Gateway, event processing, surveillance/notifications, and audit/compliance.
- [x] Prometheus alert rules cover local service availability, gateway failures/latency, event failures, DLQ events, notifications, and audit ingestion.
- [x] SLO-oriented docs and an observability runbook are included.
- [x] Correlation IDs flow through gateway requests.
- [x] Correlation IDs are propagated through key Kafka events, DLQ records, logs, and audit persistence.
- [ ] Add distributed tracing.
- [ ] Add Alertmanager routing, ownership, and escalation policies.

## Logging

- [x] Services use structured or framework logs.
- [x] Demo scripts and smoke tests print clear progress.
- [ ] Standardize log fields across all languages.
- [ ] Centralize logs in a searchable backend.
- [ ] Define log retention and PII handling.

## Metrics

- [x] HTTP, Kafka, surveillance, risk, order, portfolio, market, and notification metrics are exposed where implemented.
- [x] Prometheus scrape targets cover all backend services.
- [x] SLO-oriented dashboards are included for local demo workflows.
- [x] Service and workflow alert thresholds are documented for local demos.
- [ ] Add business KPI dashboards for trading workflows.
- [ ] Add Kafka consumer lag metrics.

## Health And Readiness

- [x] All backend services expose `/health`.
- [x] All backend services expose `/ready`.
- [x] Docker Compose healthchecks are configured.
- [x] Smoke tests cover direct and gateway health checks.
- [ ] Add dependency-specific readiness detail in every service.
- [ ] Add graceful shutdown validation under load.

## Testing

- [x] Go service tests exist for domain/service logic.
- [x] API Gateway Jest tests cover proxy behavior.
- [x] Bash syntax checks are part of release validation.
- [x] Docker Compose config validation is documented.
- [ ] Add full end-to-end automated test with containers.
- [ ] Add contract tests for Kafka event payloads.
- [ ] Add load and resilience tests.

## Security

- [x] JWT validation protects sensitive APIs.
- [x] Password auth is isolated in identity-service.
- [x] Local secrets are documented and ignored.
- [ ] Add TLS termination for production.
- [ ] Add dependency scanning in CI.
- [ ] Add container image scanning.
- [ ] Review CORS, headers, and rate-limit policies for production.

## Docker Compose

- [x] Compose starts the full local platform.
- [x] Compose includes infrastructure dependencies.
- [x] Compose config validation passes.
- [x] Compose mounts Prometheus alert rules and Grafana dashboard provisioning for local observability demos.
- [x] Backup, restore, archive, and replay scripts operate through Docker Compose.
- [ ] Compose is not a production orchestrator.
- [ ] Add backup/restore runbooks for stateful services.
- [ ] Add persistent production alert routing and dashboard ownership.

## Kubernetes / Helm

- [x] Optional Helm chart exists for application service deployment readiness.
- [x] Chart renders Deployments, Services, ConfigMap, example Secret, ServiceAccount, and optional ingress.
- [x] Liveness/readiness probes are mapped to `/health` and `/ready`.
- [x] Resource requests and limits are included.
- [x] ConfigMap and Secret responsibilities are separated.
- [x] Helm validation script is included and skips gracefully when Helm is unavailable.
- [ ] Validate chart in a real kind/minikube cluster.
- [ ] Add production secrets management, TLS ingress, autoscaling, network policies, and rollout strategy.
- [ ] Use managed or separately operated PostgreSQL, Redis, Kafka/Redpanda, MQTT, Prometheus, and Grafana for production.

## Deployment Gaps

- [x] Optional Kubernetes/Helm deployment-readiness artifacts exist.
- [ ] No production ingress/TLS setup exists.
- [ ] No CI/CD pipeline is documented as a hard requirement yet.
- [ ] No real cloud database, Kafka, or secret manager integration exists.
- [ ] No blue/green or canary release process exists.

## Known Limitations

- Local Compose uses a shared PostgreSQL database for convenience.
- Event payload schemas are documented by examples, not enforced by a schema registry.
- Notification email delivery is mock/log-only.
- Webhook delivery is intentionally simple.
- Surveillance consumes some event types that do not trigger rules yet.
- Frontend applications are placeholders/foundations, not complete production UIs.

## Future Kubernetes/Helm Work

- Add service Deployments, Services, ConfigMaps, and Secrets.
- Add Helm chart values for local, staging, and production.
- Add readiness/liveness probes mapped to current service endpoints.
- Add horizontal scaling and resource requests/limits.
- Add Kafka/PostgreSQL/Redis managed-service integration.
- Add ingress, TLS, and production identity provider integration.
