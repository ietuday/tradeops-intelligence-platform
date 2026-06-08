# Kubernetes And Helm Deployment Readiness

TradeOps remains a Docker Compose-first local demo platform. The Helm chart under `infrastructure/helm/tradeops-platform/` is an optional deployment-readiness layer for explaining how the application services could be packaged for Kubernetes.

## Deployment Architecture

The chart renders Kubernetes objects for the application services:

- `api-gateway`
- `identity-service`
- `market-data-service`
- `order-service`
- `portfolio-service`
- `strategy-service`
- `risk-engine-service`
- `surveillance-service`
- `notification-service`
- `audit-service`

Each service gets a Deployment and ClusterIP Service. Shared non-secret config lives in a ConfigMap. The JWT secret is represented by an example Kubernetes Secret with a placeholder value.

## Chart Structure

```text
infrastructure/helm/tradeops-platform/
  Chart.yaml
  values.yaml
  README.md
  templates/
    _helpers.tpl
    namespace.yaml
    configmap.yaml
    secret.example.yaml
    serviceaccount.yaml
    deployment.yaml
    service.yaml
    ingress.yaml
    NOTES.txt
```

## Values Overview

- `global.namespace`: target namespace, default `tradeops`.
- `global.imagePullPolicy`: image pull policy, default `IfNotPresent`.
- `services.*`: service enablement, image, tag, port, and replicas.
- `config.*`: database, Redis, Kafka, MQTT, JWT secret reference, and gateway timeout settings.
- `secrets.createExample`: renders a placeholder secret when true.
- `ingress.enabled`: optional API Gateway ingress, disabled by default.
- `resources`: default requests and limits for every application service.
- `probes`: liveness/readiness probe defaults.

## Service Discovery

The chart creates DNS-friendly service names such as `api-gateway`, `identity-service`, and `audit-service`. The API Gateway receives internal service URLs from the ConfigMap, for example:

```text
IDENTITY_SERVICE_URL=http://identity-service:8084
AUDIT_SERVICE_URL=http://audit-service:8092
```

Infrastructure dependencies are expected to be reachable by the names in `values.yaml`, such as `postgres`, `redis`, `redpanda`, and `mosquitto`.

## Health And Readiness Probes

Every Deployment includes:

- Liveness probe: `/health`
- Readiness probe: `/ready`
- `terminationGracePeriodSeconds: 30`

These mirror the platform’s existing health/readiness conventions.

## Resource Limits

The chart defines modest defaults:

```yaml
requests:
  cpu: 100m
  memory: 128Mi
limits:
  cpu: 500m
  memory: 512Mi
```

Tune these after load testing; they are local readiness defaults, not production sizing.

## Secret Handling

The chart uses an example placeholder:

```text
JWT_SECRET=replace-me
```

Do not use placeholder secrets for real deployments. Production should use Kubernetes Secrets managed by a secure process, External Secrets, Vault, or a cloud secret manager.

## Local Cluster Usage

With kind or minikube, build/tag images for the cluster, then render or install:

```bash
./scripts/validate-helm.sh
helm template tradeops infrastructure/helm/tradeops-platform
helm install tradeops infrastructure/helm/tradeops-platform -n tradeops --create-namespace
helm uninstall tradeops -n tradeops
```

Docker Compose remains easier for the full local demo because it already includes PostgreSQL, Redis, Redpanda, Mosquitto, Prometheus, and Grafana.

## Production Considerations

- Use managed PostgreSQL/Redis/Kafka or separately operated stateful services.
- Configure real secrets and secret rotation.
- Configure TLS ingress.
- Add migration strategy and backup/restore workflow.
- Add autoscaling, pod disruption budgets, network policies, and rollout strategy.
- Add log aggregation, tracing, and Alertmanager routing.

## Known Limitations

- The chart does not install full stateful infrastructure.
- The chart does not provision Kafka topics.
- The chart does not include production TLS, cloud ingress, or managed secrets.
- The default image tags are local placeholders.

