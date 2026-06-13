# Helm Chart

The v3 chart is an umbrella chart:

```text
deployments/helm/tradeops
```

It includes Deployments, Services, ConfigMaps, Secret references, optional inline development Secrets, Ingress, HPAs, PDBs, NetworkPolicies, migration and seed Jobs, optional ServiceMonitor resources, and an optional OpenTelemetry Collector.

Validate:

```bash
./scripts/k8s-validate.sh
helm template tradeops deployments/helm/tradeops -f deployments/helm/tradeops/values-local.yaml
```

The chart uses stable service names such as `api-gateway`, `identity-service`, `order-service`, and `trading-dashboard-react` so the API Gateway can use Kubernetes DNS names directly.

