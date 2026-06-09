# Deployment Readiness Checklist

Use this checklist before trying the optional Helm chart in a local Kubernetes cluster.

## Application Images

- [ ] API Gateway image is built and tagged.
- [ ] Go service images are built and tagged.
- [ ] Python service images are built and tagged.
- [ ] Image tags in `values.yaml` match available images.
- [ ] Local cluster can access those images.

## Configuration

- [ ] Environment values are reviewed.
- [ ] Service URLs resolve inside the cluster.
- [ ] Database URL is configured.
- [ ] Redis endpoint is configured.
- [ ] Kafka/Redpanda brokers are configured.
- [ ] MQTT broker URL is configured.

## Secrets

- [ ] Placeholder `JWT_SECRET=replace-me` is replaced before any real deployment.
- [ ] Secrets are managed with Kubernetes Secrets, External Secrets, Vault, or a cloud secret manager.
- [ ] Secret rotation and access ownership are documented.

## Dependencies

- [ ] PostgreSQL is available and migrations are planned.
- [ ] Redis is available.
- [ ] Redpanda/Kafka is available.
- [ ] Mosquitto or equivalent MQTT source is available.
- [ ] Prometheus/Grafana are configured if Kubernetes observability is needed.

## Kubernetes Runtime

- [ ] Namespace exists or Helm creates it.
- [ ] Deployments render successfully with `helm template`.
- [ ] Probes are enabled and mapped to `/health` and `/ready`.
- [ ] Resource requests and limits are set.
- [ ] Resource requests/limits are compared against local benchmark observations.
- [ ] Ingress is configured only when needed.
- [ ] Logs are checked after startup.

## Performance And Scaling

- [ ] Local perf smoke has been run with `./scripts/perf-smoke.sh`.
- [ ] Optional k6 scenario has been run for the API Gateway before scaling changes.
- [ ] Stateful bottlenecks are reviewed before increasing application replicas.
- [ ] PostgreSQL, Redpanda/Kafka, and Redis capacity plans are documented.
- [ ] SLO targets are used to decide whether scaling is needed.

## Rollback And Safety

- [ ] Docker Compose remains available as the primary local demo runtime.
- [ ] Helm uninstall command is known.
- [ ] Database backup exists before risky deployment tests.
- [ ] Rollback plan is documented.
