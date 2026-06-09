# Security Checklist

## Local Development

- [ ] Copy `.env.example` to local `.env` files only.
- [ ] Use local-only JWT secrets and database credentials.
- [ ] Keep Docker Compose bound to local/demo ports.
- [ ] Run `./scripts/security-check.sh` before publishing.

## Before Commit

- [ ] Check `git status --short` for accidental secrets or generated files.
- [ ] Confirm no real `.env`, `.pem`, `.key`, `id_rsa`, or `private_key` files are tracked.
- [ ] Confirm docs contain placeholders only.
- [ ] Run unit tests for modified services.

## Before Demo

- [ ] Start with `make dev-up`.
- [ ] Run `make smoke`.
- [ ] Use a demo token with the correct role.
- [ ] Confirm Prometheus/Grafana are local-only.

## Before Deployment

- [ ] Replace local secrets with environment-managed secrets.
- [ ] Configure production CORS origins.
- [ ] Set appropriate API Gateway rate limits and body limits.
- [ ] Add TLS ingress and access controls.
- [ ] Review RBAC per endpoint.

## API Gateway

- [x] Helmet security headers enabled.
- [x] `x-powered-by` disabled.
- [x] Request body limit configured.
- [x] Rate limit configured.
- [x] Correlation ID returned on security errors.

## JWT/RBAC

- [ ] Confirm every protected API validates JWTs.
- [ ] Confirm lifecycle and export actions require elevated roles.
- [ ] Confirm demo users have least privilege for the scenario.

## Secrets

- [ ] Use `.env.example` for placeholders only.
- [ ] Do not commit local `.env` files.
- [ ] Rotate demo secrets before sharing environments.

## Docker Compose

- [ ] Treat Compose as local-only.
- [ ] Validate with `docker compose -f infrastructure/docker/docker-compose.yml config`.
- [ ] Keep Redpanda, Postgres, Redis, Mosquitto, Prometheus, and Grafana local unless explicitly secured.

## Helm/Kubernetes

- [ ] Replace placeholder Secrets.
- [ ] Add TLS ingress, network policies, autoscaling, and managed stateful dependencies for real production.
- [ ] Validate chart rendering with `./scripts/validate-helm.sh`.

## Observability

- [ ] Keep `/metrics` endpoints scoped to trusted networks.
- [ ] Use correlation IDs in demos and incident walkthroughs.
- [ ] Review dashboards for sensitive labels before public screenshots.

## Audit

- [ ] Restrict audit read/export APIs to elevated roles.
- [ ] Preserve `correlationId` on security-sensitive actions.
- [ ] Validate audit retention expectations before production use.

## Backup/Replay Safety

- [ ] Treat backups as sensitive.
- [ ] Replay only known-good sample events unless the root cause is understood.
- [ ] Keep destructive restore/delete actions behind explicit confirmation flags.
