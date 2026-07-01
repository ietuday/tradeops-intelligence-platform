# Order Outbox Runbook

## Is The Publisher Running?

Check `GET /internal/outbox/status` on the Order Service or inspect logs for `outbox publisher started`. Prometheus should expose `tradeops_outbox_claimed_total`.

## Inspect Backlog

Use metrics:

```promql
tradeops_outbox_pending
tradeops_outbox_processing
tradeops_outbox_failed
tradeops_outbox_oldest_pending_age_seconds
```

SQL:

```sql
SELECT status, count(*) FROM order_outbox GROUP BY status ORDER BY status;
SELECT id, event_type, topic, created_at, attempt_count, last_error
FROM order_outbox
WHERE published_at IS NULL AND status = 'pending'
ORDER BY created_at, id
LIMIT 20;
```

## Terminal Failures

```sql
SELECT id, event_type, topic, failed_at, attempt_count, last_error
FROM order_outbox
WHERE status = 'failed'
ORDER BY failed_at DESC
LIMIT 50;
```

Do not manually set `published_at` unless you are deliberately suppressing publication and have recorded the operational decision.

## Redpanda Unavailable

Rows remain unpublished, attempts increase, `available_at` moves forward with backoff, and the service stays alive. After Redpanda recovers, claimable rows publish normally.

## Retry A Failed Row

After confirming the payload is valid and the consumer can tolerate duplicates:

```sql
UPDATE order_outbox
SET status = 'pending',
    failed_at = NULL,
    available_at = now(),
    last_error = NULL
WHERE id = '<outbox-id>' AND published_at IS NULL;
```

## Stuck Lease

A lease is stale when `status='processing'`, `published_at IS NULL`, and `lease_until < now()`. Another replica should recover it on the next poll.

## Alerts

Recommended alert signals:

- `tradeops_outbox_pending` remains above the environment threshold.
- `tradeops_outbox_oldest_pending_age_seconds` exceeds the SLA.
- `rate(tradeops_outbox_publish_errors_total[5m])` is elevated.
- `tradeops_outbox_failed > 0`.
- Backlog exists and `tradeops_outbox_published_total` is not increasing.

## Scaling

Scale Order Service replicas when backlog age rises and Kafka/database latency is healthy. Multiple replicas are safe because claims use PostgreSQL row locks and leases.

## Disable Publishing

Set `OUTBOX_ENABLED=false` only for controlled maintenance. Outbox rows continue to accumulate and can be published after re-enabling.
