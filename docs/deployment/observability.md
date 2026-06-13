# Observability

The chart preserves the Compose observability model:

```mermaid
flowchart LR
  App[Applications] --> Metrics[/metrics]
  Metrics --> Prometheus[Prometheus]
  Prometheus --> Grafana[Grafana]
  App --> OTLP[OTLP HTTP/gRPC]
  OTLP --> Collector[OpenTelemetry Collector]
  Collector --> Jaeger[Jaeger or OTLP Backend]
```

Portable Prometheus annotations are enabled by default. `ServiceMonitor` resources are optional because they require Prometheus Operator CRDs.

The OpenTelemetry Collector is optional. Local demos may send directly to Jaeger or to the collector; production should use an organization-managed tracing backend.

