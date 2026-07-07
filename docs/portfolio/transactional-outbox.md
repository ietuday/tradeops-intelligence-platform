# Portfolio Transactional Outbox

v3.3.0 moves `portfolio.updated` publication behind a Portfolio Service transactional outbox.

When a `trade.executed` event changes portfolio state, the service now commits these records in one PostgreSQL transaction:

- buyer/seller cash and holdings updates
- portfolio transaction ledger rows
- processed event idempotency record
- portfolio snapshot row
- `portfolio.updated` outbox row

The background publisher reads `portfolio_outbox_events`, marks rows as `publishing`, publishes to Kafka, then marks rows `published`. Publish failures are retried with exponential backoff until `PORTFOLIO_OUTBOX_MAX_ATTEMPTS`; terminal failures remain in the table with `status='failed'`.

## Configuration

- `PORTFOLIO_OUTBOX_ENABLED=true`
- `PORTFOLIO_OUTBOX_POLL_INTERVAL=2s`
- `PORTFOLIO_OUTBOX_BATCH_SIZE=100`
- `PORTFOLIO_OUTBOX_MAX_ATTEMPTS=10`
- `PORTFOLIO_OUTBOX_BASE_BACKOFF=1s`
- `PORTFOLIO_OUTBOX_MAX_BACKOFF=60s`

## Status And Metrics

Status:

```bash
curl http://localhost:8087/internal/portfolio/outbox/status
```

Metrics:

- `tradeops_portfolio_outbox_events_total{status}`
- `tradeops_portfolio_outbox_publish_attempts_total{event_type,status}`
- `tradeops_portfolio_outbox_publish_errors_total{event_type,reason}`
- `tradeops_portfolio_outbox_pending_count`
- `tradeops_portfolio_outbox_oldest_pending_age_seconds`

## Known Limitations

- `portfolio.snapshot.created` remains a database snapshot record; v3.3.0 focuses reliable Kafka publication on `portfolio.updated`.
- Kafka delivery is at-least-once. Consumers must remain idempotent.
