# Transactional Outbox Publisher

v3.1.1 moves Order Service event delivery to a transactional outbox publisher.

## Delivery Semantics

Database state and event intent are committed atomically. Kafka delivery is at-least-once. Consumers must be idempotent.

This is not exactly-once delivery. If Kafka accepts an event and the service crashes before `order_outbox.published_at` is updated, the row can be republished after lease expiry.

## Transaction Boundary

Order lifecycle updates, execution persistence, `order_events`, and `order_outbox` inserts happen in one PostgreSQL transaction. The background publisher only reads committed `order_outbox` rows and publishes the stored JSON payload exactly as persisted.

## Claim And Lease Design

The publisher claims rows with `FOR UPDATE SKIP LOCKED`, ordered by `created_at, id`. It marks rows `processing`, sets `locked_by`, `locked_at`, and `lease_until`, commits, then publishes outside the database transaction.

Rows are claimable when they are unpublished, below max attempts, available for retry, and either unlocked or past `lease_until`. This supports multiple Order Service replicas and crash recovery without holding row locks during Kafka I/O.

## Retry And Failure

Transient publication errors increment `attempt_count`, clear ownership, write a sanitized `last_error`, and set `available_at` using exponential backoff with jitter. Malformed JSON is terminal because retrying cannot make it valid.

Rows that reach `OUTBOX_MAX_ATTEMPTS` are marked `status='failed'` with `failed_at` and remain available for operational inspection. Failed rows are never deleted automatically.

## Kafka Metadata

The Kafka key is `tenantID:aggregateID`, so events for the same order stay on the same partition. Headers include `event-id`, `event-type`, `tenant-id`, `correlation-id`, `traceparent`, `tracestate`, and `content-type=application/json` when present.

This slice publishes each claimed batch sequentially. That keeps events for one aggregate in database order. `OUTBOX_PUBLISH_CONCURRENCY` is validated and reserved for a keyed-concurrency implementation where different aggregates can publish in parallel without reordering events for one order.

Global ordering across different orders is not guaranteed or required.

## Observability

Metrics use the `tradeops_outbox_*` prefix and avoid high-cardinality labels. The publisher creates OpenTelemetry spans for poll, publish, and mark operations. The internal status endpoint is `GET /internal/outbox/status`.

## Configuration

Key settings include `OUTBOX_ENABLED`, `OUTBOX_POLL_INTERVAL`, `OUTBOX_BATCH_SIZE`, `OUTBOX_PUBLISH_TIMEOUT`, `OUTBOX_SHUTDOWN_TIMEOUT`, `OUTBOX_LEASE_DURATION`, `OUTBOX_BASE_BACKOFF`, `OUTBOX_MAX_BACKOFF`, `OUTBOX_MAX_ATTEMPTS`, and `OUTBOX_ERROR_MAX_LENGTH`.

`OUTBOX_LEASE_DURATION` must be greater than `OUTBOX_PUBLISH_TIMEOUT`.
