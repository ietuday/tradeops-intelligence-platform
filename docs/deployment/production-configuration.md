# Production Configuration

Use `values-production.yaml` as a starting point, not a universal production profile.

Production deployments should set:

- `secrets.existingSecretName`
- external PostgreSQL, Redis, Kafka/Redpanda, MQTT, and tracing endpoints
- ingress hosts and TLS secret names
- resource requests and limits based on measured load
- autoscaling ranges
- NetworkPolicy mode compatible with the cluster CNI
- managed backup and restore plans

Example:

```bash
helm upgrade --install tradeops deployments/helm/tradeops \
  --namespace tradeops \
  --create-namespace \
  -f deployments/helm/tradeops/values-production.yaml \
  --set secrets.existingSecretName=tradeops-production-secrets \
  --set postgresql.external.host=postgres.internal.example \
  --set redis.external.host=redis.internal.example \
  --set kafka.external.brokers=broker-0:9092,broker-1:9092 \
  --wait
```

Do not enable inline Secrets or demo seed data in production.

