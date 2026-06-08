# TradeOps Platform Helm Chart

This chart is an optional Kubernetes deployment-readiness layer for TradeOps Intelligence Platform. Docker Compose remains the recommended local demo runtime.

## What This Chart Includes

- Namespace manifest.
- ConfigMap for shared non-secret configuration.
- Example Kubernetes Secret with placeholder `JWT_SECRET=replace-me`.
- Deployments and ClusterIP Services for all application services.
- Liveness and readiness probes for `/health` and `/ready`.
- Resource requests and limits.
- Prometheus scrape annotations.
- Optional ingress for `api-gateway`.

## What This Chart Does Not Include

- Production PostgreSQL, Redis, Redpanda, Mosquitto, Prometheus, or Grafana installs.
- Cloud-specific storage, ingress, load balancers, or secret managers.
- Real secrets.
- Production-grade autoscaling, pod disruption budgets, network policies, or service mesh config.

For production, use managed or separately operated infrastructure dependencies.

## Validate Locally

```bash
helm lint infrastructure/helm/tradeops-platform
helm template tradeops infrastructure/helm/tradeops-platform
./scripts/validate-helm.sh
```

If Helm is not installed, `scripts/validate-helm.sh` skips gracefully and explains how to validate later.

## Install

```bash
helm install tradeops infrastructure/helm/tradeops-platform -n tradeops --create-namespace
```

## Uninstall

```bash
helm uninstall tradeops -n tradeops
```

## Image Assumptions

The default values use local image tags such as `tradeops/api-gateway:local`. Build and tag images for your local cluster before installing, or override image repositories/tags in `values.yaml`.

## Secret Handling

The chart renders an example secret by default:

```yaml
JWT_SECRET: "replace-me"
```

Do not use this placeholder for real deployments. Use Kubernetes Secrets, External Secrets, Vault, or a cloud secret manager.

## Dependency Assumptions

The application services expect these dependencies to be reachable by DNS names from `values.yaml`:

- `postgres`
- `redis`
- `redpanda`
- `mosquitto`

For local Kubernetes, you can run equivalent services in-cluster or point the values at externally reachable endpoints.

## Production Gaps

- No managed database, backup, or migration workflow is included.
- No Kafka topic provisioning is included.
- No TLS or production ingress configuration is included.
- No Alertmanager, log aggregation, or tracing stack is included.
- No autoscaling or rollout strategy is included.

