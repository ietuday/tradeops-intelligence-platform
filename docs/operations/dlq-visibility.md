# DLQ Visibility

Dead-letter queues preserve events that could not be processed after retry. v3.4.0 makes DLQ presence visible through metrics and internal status without adding replay or destructive operations.

## Topic Convention

Use the configured DLQ topic when a service already has one. Otherwise the convention is:

```text
<original-topic>.dlq
```

Portfolio Service uses its existing `PORTFOLIO_TRADE_DLQ_TOPIC` and exposes it through `/internal/consumers/status`.

## Metrics

```text
tradeops_dlq_messages{service,topic}
tradeops_dlq_oldest_message_age_seconds{service,topic}
tradeops_dlq_events_total{service,topic,event_type,reason}
```

`reason` values must stay bounded, for example `processing_error` or `publish_error`. Payload content and IDs are not metric labels.

## Status Rules

`healthy` means the service has not observed DLQ messages.

`degraded` means DLQ messages are present and require investigation.

`critical` means the oldest known DLQ message age is above `DLQ_OLDEST_AGE_CRITICAL`.

Configuration:

```text
DLQ_OBS_ENABLED=true
DLQ_TOPIC_SUFFIX=.dlq
DLQ_OLDEST_AGE_WARN=5m
DLQ_OLDEST_AGE_CRITICAL=30m
```

## PromQL

DLQ messages by service and topic:

```promql
sum by (service, topic) (tradeops_dlq_messages)
```

Oldest DLQ age:

```promql
max by (service, topic) (tradeops_dlq_oldest_message_age_seconds)
```

New DLQ event rate:

```promql
sum by (service, topic, event_type, reason) (rate(tradeops_dlq_events_total[5m]))
```

## Operations Flow

1. Confirm the status endpoint reports DLQ messages.
2. Inspect service logs with the same time range.
3. Look at `tradeops_consumer_processing_errors_total` for the bounded failure reason.
4. Inspect the DLQ topic with read-only Kafka tooling.
5. Fix the parser, schema, dependency, or data issue.
6. Replay only with approved replay tooling and known-good payloads.

v3.4.0 deliberately does not consume, delete, or replay DLQ records from the status endpoint.
