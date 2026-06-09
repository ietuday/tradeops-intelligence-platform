# OpenTelemetry Runbook

Use this when traces are missing or the Jaeger demo does not look right.

## No Traces In Jaeger

1. Confirm `OTEL_ENABLED=true`.
2. Confirm Jaeger is running: `curl -fsS http://localhost:16686`.
3. Confirm services use `OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318` in Compose.
4. Generate traffic through the API Gateway.

## Service Not Listed

1. Hit that service through the gateway or directly.
2. Wait a few seconds for batch export.
3. Check service logs for exporter connection errors.
4. Confirm the service is one of the v2.3.0 instrumented services.

## Trace Breaks At Kafka Boundary

Kafka tracing is lightweight in v2.3.0. Confirm the source event includes `traceparent` as a Kafka header or JSON field. If a manually published event lacks it, the consumer starts a new trace but keeps `correlationId`.

## High Trace Volume

Lower sampling for local stress tests:

```env
OTEL_TRACES_SAMPLER_ARG=0.1
```

Do not add trace IDs or correlation IDs as Prometheus labels.

## Jaeger Unavailable

The services continue to run if Jaeger is unavailable. Check logs for exporter errors, then restart Jaeger:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml up -d jaeger
```

## Trace Exists But Logs Not Found

Jaeger search uses `traceId`; logs and audit search use `correlationId`. Find the `correlation.id` span attribute in Jaeger, then grep logs:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml logs | grep <correlation-id>
```

## Correlate Trace ID With Correlation ID

1. Open the trace in Jaeger.
2. Inspect span tags for `correlation.id`.
3. Use that value in logs, DLQ payloads, and audit APIs.
