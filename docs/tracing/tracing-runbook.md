# Correlation Tracing Runbook

Use this runbook to trace a single request or event flow with `X-Correlation-ID` / `correlationId`.

For Jaeger/OpenTelemetry traces, see [OpenTelemetry tracing](opentelemetry.md) and [OpenTelemetry runbook](otel-runbook.md). Use `traceId` in Jaeger and `correlationId` in logs/audit queries.

## Trace Order Creation To Surveillance Alert

Starting point: a client creates an order with `X-Correlation-ID`.

Command:

```bash
CORRELATION_ID=demo-correlation-123 ./scripts/demo-correlation-tracing.sh --publish-sample
```

Expected logs:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway order-service surveillance-service | grep demo-correlation-123
```

Prometheus query:

```text
sum(rate(surveillance_alerts_created_total[5m])) or vector(0)
```

## Trace Surveillance Alert To Notification

Starting point: `surveillance.alert.created` event.

Command:

```bash
CORRELATION_ID=demo-correlation-123 ./scripts/replay-sample-events.sh --notifications
```

Expected logs:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service notification-service | grep demo-correlation-123
```

## Trace Failed Event To DLQ

Starting point: retry/DLQ metrics or service error logs.

Command:

```bash
./scripts/replay-dlq-events.sh --topic notification.dlq --dry-run
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume notification.dlq -n 1
```

Expected record: DLQ payload includes `correlationId` when the failed source event or Kafka header had one.

## Trace Audit Log By Correlation ID

Starting point: known correlation ID.

Command:

```bash
curl "http://localhost:8080/api/audit/logs?correlationId=demo-correlation-123" \
  -H "Authorization: Bearer ${TOKEN}"
```

Expected result: matching audit logs where `audit_logs.correlation_id` equals the correlation ID.

## Trace API Gateway Proxy Error

Starting point: `502` or `504` from the gateway.

Command:

```bash
curl -i http://localhost:8080/api/orders/health \
  -H "X-Correlation-ID: demo-correlation-123"
```

Expected response: body and response header include the same correlation ID.

## Trace Webhook Delivery Failure

Starting point: notification delivery failure or webhook timeout.

Commands:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs notification-service | grep demo-correlation-123
curl http://localhost:8091/metrics | grep notification_delivery_failures_total
```

Expected result: notification logs and metrics show the failed delivery path; audit logs may also contain notification lifecycle events with the same correlation ID.
