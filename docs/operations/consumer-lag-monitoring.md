# Consumer Lag Monitoring

v3.4.0 adds consumer observability for event-driven services, starting with Portfolio Service. The goal is operational visibility, not business behavior changes.

Consumer lag is the distance between the latest Kafka/Redpanda offset on a topic partition and the offset committed by a consumer group. In a trading workflow, lag means downstream state may be stale: executions can be published while portfolio positions, risk exposure, alerts, or notifications have not caught up yet.

## Signals

Portfolio Service exposes:

```text
GET /internal/consumers/status
```

The response contains a UTC `checkedAt`, overall status, per-consumer status, partition lag fields, processing timestamps, consecutive errors, and DLQ status. The endpoint does not include message payloads.

Key Prometheus metrics:

```text
tradeops_consumer_lag_messages{service,consumer_group,topic,partition}
tradeops_consumer_lag_oldest_age_seconds{service,consumer_group,topic}
tradeops_consumer_last_processed_timestamp_seconds{service,consumer_group,topic}
tradeops_consumer_processing_total{service,consumer_group,topic,event_type,status}
tradeops_consumer_processing_errors_total{service,consumer_group,topic,event_type,reason}
tradeops_consumer_rebalances_total{service,consumer_group,topic}
```

Labels are intentionally bounded. Do not add event IDs, order IDs, user IDs, request IDs, correlation IDs, or payload fields as metric labels.

## Status Rules

`healthy` means lag is below the warning threshold and processing has not accumulated repeated errors.

`degraded` means lag crossed the warning or critical threshold, or the consumer has repeated processing errors.

`stalled` means lag exists and processing has not progressed for `CONSUMER_OBS_STALLED_AFTER`.

`unknown` means the service cannot confidently report offset state, such as before the consumer starts or while broker metadata is unavailable.

Liveness and readiness should not fail only because lag is high. Lag is usually a scaling or downstream-processing signal. Kubernetes should not restart pods simply because a backlog exists.

## Configuration

```text
CONSUMER_OBS_ENABLED=true
CONSUMER_OBS_POLL_INTERVAL=10s
CONSUMER_OBS_LAG_WARN_THRESHOLD=1000
CONSUMER_OBS_LAG_CRITICAL_THRESHOLD=10000
CONSUMER_OBS_STALLED_AFTER=2m
CONSUMER_OBS_MAX_TOPIC_SCAN=100
```

The Portfolio implementation is safe if broker metadata is unavailable. The current status endpoint keeps reporting the latest in-memory consumer health and uses bounded metrics.

## PromQL

Lag by service, group, and topic:

```promql
sum by (service, consumer_group, topic) (tradeops_consumer_lag_messages)
```

Oldest lag age:

```promql
max by (service, consumer_group, topic) (tradeops_consumer_lag_oldest_age_seconds)
```

Processing error rate:

```promql
sum by (service, consumer_group, topic, event_type, reason) (rate(tradeops_consumer_processing_errors_total[5m]))
```

Last processed age:

```promql
time() - max by (service, consumer_group, topic) (tradeops_consumer_last_processed_timestamp_seconds)
```

## Troubleshooting

1. Check `/internal/consumers/status` on the affected service.
2. Compare lag metrics with processing error metrics.
3. Check DLQ metrics and service logs for bounded reason codes.
4. Confirm Redpanda/Kafka is reachable and the source topic has partitions.
5. If lag grows but errors are zero, scale consumers or inspect downstream storage latency.
6. If lag grows with errors, fix the processing failure before scaling.

## Current Scope

Portfolio Service is wired first because it is the highest-impact downstream consumer for executions. Shared metric names and status types are reusable by order, surveillance, notification, and audit services as their consumer implementations are upgraded.
