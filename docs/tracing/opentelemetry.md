# OpenTelemetry Distributed Tracing

v2.3.0 adds local-demo OpenTelemetry tracing while preserving the existing `X-Correlation-ID` model.

## What It Adds

OpenTelemetry provides span timing and distributed trace visualization for the key request/event path:

```text
Client -> API Gateway -> Order Service -> Kafka order.created
  -> Surveillance Service -> Kafka surveillance.alert.created
  -> Notification Service -> Kafka notification.created/notification.sent
  -> Audit Service
```

Correlation IDs remain the stable business/debug identifier for logs, Kafka payloads, DLQ records, and audit lookup.

## Local Jaeger Setup

Docker Compose includes Jaeger all-in-one:

```bash
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml up -d jaeger
```

Open:

```text
Jaeger UI: http://localhost:16686
OTLP HTTP: http://localhost:4318
OTLP gRPC: http://localhost:4317
```

## Instrumented Services

- `api-gateway`
- `order-service`
- `surveillance-service`
- `notification-service`
- `audit-service`

Python services are documented as a future extension.

## Propagation

HTTP uses W3C headers:

- `traceparent`
- `tracestate`

The API Gateway forwards those headers with:

- `X-Correlation-ID`
- `X-Tenant-ID`

Kafka propagation is lightweight and additive. Selected producers include `traceparent` as a Kafka header and JSON field when an active span exists. Consumers extract `traceparent` from headers or payload and start `consume <topic>` spans.

v2.5.0 event schemas treat `traceparent` as optional metadata alongside `eventVersion`, `tenantId`, and `correlationId`. See [event envelope](../events/event-envelope.md) for the standard event shape.

## Correlation ID Vs Trace ID

| ID | Purpose | Where to search |
| --- | --- | --- |
| `correlationId` | Stable app-level debug/business flow ID | logs, events, DLQ, audit APIs |
| `traceId` | OpenTelemetry trace tree ID | Jaeger |
| `spanId` | One operation inside a trace | Jaeger span details |

Do not use `traceId`, `spanId`, `correlationId`, or unbounded `tenantId` values as Prometheus labels.

## Configuration

```env
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_SERVICE_VERSION=2.3.0
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=1.0
```

Tracing defaults to disabled in code unless `OTEL_ENABLED=true`.

## Demo

```bash
./scripts/demo-otel-tracing.sh
TOKEN=<jwt> ./scripts/demo-otel-tracing.sh --create-order
```

In Jaeger, search service `api-gateway` and operations such as `GET /api/orders/health` or `POST /api/orders`.

Admin operations routes add span attributes for `admin.endpoint`, `tenant.id`, and `correlation.id` when tracing is enabled. This keeps `/api/admin` health and summary calls visible without adding high-cardinality metric labels.

## Known Limitations

- Kafka propagation uses payload/header `traceparent`; full Kafka header instrumentation is future work.
- Database spans are not explicitly instrumented in this pass.
- Python services are not instrumented yet.
- Audit logs persist `correlationId`; trace IDs are present in events where practical but are not the primary audit lookup key.
