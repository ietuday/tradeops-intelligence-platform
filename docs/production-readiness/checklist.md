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
- [ ] Add schema validation or schema registry contracts.
- [ ] Define dead-letter topics and replay procedures.
- [ ] Add consumer lag monitoring.

## Observability

- [x] Services expose Prometheus metrics.
- [x] Docker Compose includes Prometheus and Grafana.
- [x] Basic Grafana dashboard exports are included.
- [x] Correlation IDs flow through gateway requests.
- [ ] Add distributed tracing.
- [ ] Add alert rules for service health, latency, and event failures.

## Logging

- [x] Services use structured or framework logs.
- [x] Demo scripts and smoke tests print clear progress.
- [ ] Standardize log fields across all languages.
- [ ] Centralize logs in a searchable backend.
- [ ] Define log retention and PII handling.

## Metrics

- [x] HTTP, Kafka, surveillance, risk, order, portfolio, market, and notification metrics are exposed where implemented.
- [x] Prometheus scrape targets cover all backend services.
- [ ] Add SLO-oriented dashboards.
- [ ] Add service-specific alert thresholds.
- [ ] Add business KPI dashboards for trading workflows.

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
- [ ] Compose is not a production orchestrator.
- [ ] Add backup/restore runbooks for stateful services.
- [ ] Add persistent operational dashboards and alerting rules.

## Deployment Gaps

- [ ] No Kubernetes or Helm deployment has been added yet.
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
