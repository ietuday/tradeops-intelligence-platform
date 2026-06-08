# Audit Trail

`audit-service` normalizes important platform events into searchable audit logs for compliance-style review, debugging, and interview demos.

## What It Does

- Consumes user, order, portfolio, risk, surveillance, and notification events from Redpanda/Kafka.
- Normalizes each event into an `audit_logs` PostgreSQL row.
- Publishes `audit.log.created` after a log is stored.
- Exposes searchable list, detail, summary, and export APIs.
- Uses retry/backoff and `audit.dlq` for failed event processing.

## Consumed Topics

`user.registered`, `user.login`, `user.logout`, `order.created`, `order.cancelled`, `order.filled`, `portfolio.updated`, `risk.score.updated`, `risk.breached`, `surveillance.alert.created`, `surveillance.alert.acknowledged`, `surveillance.alert.resolved`, `surveillance.alert.dismissed`, `notification.read`, `notification.failed`, `notification.sent`, `notification.retry_requested`.

## Audit Log Schema

Key fields: `event_type`, `service_name`, `actor_user_id`, `actor_role`, `entity_type`, `entity_id`, `action`, `description`, `severity`, `correlation_id`, `ip_address`, `user_agent`, `metadata`, `source_event_key`, and `created_at`.

Severity values: `INFO`, `WARNING`, `HIGH`, `CRITICAL`.

## APIs

Gateway base path: `/api/audit`

Direct service base path: `/api/v1/audit`

```bash
curl "http://localhost:8080/api/audit/logs?serviceName=order-service&limit=20" \
  -H "Authorization: Bearer ${TOKEN}"

curl "http://localhost:8080/api/audit/summary" \
  -H "Authorization: Bearer ${TOKEN}"

curl "http://localhost:8080/api/audit/export?format=csv&severity=WARNING" \
  -H "Authorization: Bearer ${TOKEN}"
```

Supported filters: `eventType`, `serviceName`, `actorUserId`, `entityType`, `entityId`, `action`, `severity`, `correlationId`, `from`, `to`, `limit`, and `offset`.

## RBAC

- Read APIs: `trading_admin`, `risk_manager`, `analyst`
- Export API: `trading_admin`, `risk_manager`
- Health, readiness, and metrics: no JWT required

## Idempotency

`audit_logs.source_event_key` is unique when present. The service builds it from `topic:eventId` when the source event has an `eventId`; otherwise it falls back to topic, action, entity ID, and timestamp. Duplicate inserts are skipped and counted with `audit_logs_duplicate_skipped_total`.

## DLQ Behavior

Failed events are retried using:

```bash
EVENT_PROCESSING_MAX_RETRIES=3
EVENT_PROCESSING_RETRY_BACKOFF_MS=500
EVENT_PROCESSING_RETRY_BACKOFF_MULTIPLIER=2
```

After retry exhaustion, the event is published to `audit.dlq` with original topic, original payload, error message, service name, failure timestamp, correlation ID, and retry count.

## Limitations

- Export returns JSON or CSV directly from the API; v1.3.0 does not create durable export files.
- Event schemas are tolerant and example-driven, not schema-registry enforced.
- Unknown topics are normalized as generic observed events.
