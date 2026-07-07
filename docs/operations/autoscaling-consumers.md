# Autoscaling Consumers

v3.4.0 adds Helm values and template support for scaling consumers from lag-related metrics. Autoscaling is disabled by default.

Consumer lag is a scaling signal when processing is healthy but throughput is too low. It is not a good scaling signal when every message fails. Check DLQ and processing error metrics before increasing replicas.

## Helm Values

Portfolio Service exposes the scaffold:

```yaml
portfolio:
  consumerAutoscaling:
    enabled: false
    minReplicas: 1
    maxReplicas: 5
    lagMetricName: tradeops_consumer_lag_messages
    lagTargetAverageValue: "1000"
    dlqMetricName: tradeops_dlq_messages
```

The HPA still renders only when the application autoscaling block is enabled:

```yaml
applications:
  portfolioService:
    autoscaling:
      enabled: true
```

## Prerequisites

Kubernetes does not read Prometheus metrics directly for HPA custom metrics. Use one of these before enabling lag scaling:

1. Prometheus Adapter configured for `tradeops_consumer_lag_messages`.
2. KEDA with a Prometheus trigger, if your cluster standardizes on KEDA.
3. A managed observability adapter that exposes the metric through the Kubernetes custom metrics API.

Local Kind deployments keep this disabled so Helm rendering does not require Prometheus Adapter or KEDA.

## Example HPA Metric

When enabled for Portfolio Service, the chart adds a Pods metric:

```yaml
metrics:
  - type: Pods
    pods:
      metric:
        name: tradeops_consumer_lag_messages
      target:
        type: AverageValue
        averageValue: "1000"
```

## Runbook

1. Confirm lag is high with Prometheus.
2. Confirm error and DLQ rates are low.
3. Enable autoscaling in a non-production environment.
4. Confirm the custom metric appears through `kubectl get --raw /apis/custom.metrics.k8s.io`.
5. Watch lag drain after replicas increase.

Do not wire liveness to lag. A lagging consumer should be scaled, debugged, or throttled upstream; restarting it in a loop usually makes recovery slower.
