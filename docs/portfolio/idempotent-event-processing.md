# Portfolio Idempotent Event Processing

Portfolio Service consumes `trade.executed` events from Order Service. Because Kafka can redeliver messages after retries, consumer group rebalances, or crashes, Portfolio Service must not apply the same execution twice.

v3.3.0 records every processed trade in `portfolio_processed_events` using a tenant-scoped idempotency key:

- `execution:<executionId>` when `executionId` is present
- `trade:<tradeId>` when no execution ID is present
- `event:<eventId>` as a fallback

For normal `trade.executed` events, `executionId` is authoritative.

## Processing Flow

1. Validate and normalize the incoming event.
2. Begin a database transaction.
3. Look up `portfolio_processed_events` by `tenant_id` and idempotency key.
4. If present, skip mutation and acknowledge the message safely.
5. If absent, update portfolio state, insert the processed-event row, insert `portfolio.updated` into `portfolio_outbox_events`, and commit.

If the outbox insert fails, the portfolio mutation rolls back with the rest of the transaction.

## Operations

Consumer status:

```bash
curl http://localhost:8087/internal/portfolio/consumer/status
```

Metrics:

- `tradeops_portfolio_events_consumed_total{event_type,status}`
- `tradeops_portfolio_events_duplicate_total{event_type}`
- `tradeops_portfolio_event_processing_errors_total{event_type,reason}`
- `tradeops_portfolio_event_processing_duration_seconds{event_type}`

## Demo

```bash
./scripts/demo-portfolio-outbox-idempotency.sh
```

The demo publishes the same `trade.executed` event twice and verifies the buyer position changes only once.
