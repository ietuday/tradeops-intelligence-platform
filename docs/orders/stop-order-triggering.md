# Stop Order Triggering

v3.2.0 adds production-style activation for pending `STOP` and `STOP_LIMIT` orders using latest reference prices.

## Behavior

Reference prices are stored in the Order Service `reference_prices` table by `tenant_id` and `symbol`. The Market Data Service publishes `market.price.updated` events, and the Order Service consumes those events to upsert the latest price.

Trigger rules:

| Side | Type | Condition |
| --- | --- | --- |
| `BUY` | `STOP` | reference price >= stop price |
| `SELL` | `STOP` | reference price <= stop price |
| `BUY` | `STOP_LIMIT` | reference price >= stop price |
| `SELL` | `STOP_LIMIT` | reference price <= stop price |

Activation:

- `STOP` becomes `MARKET`.
- `STOP_LIMIT` becomes `LIMIT` and preserves its limit price.

After activation, the Order Service reuses the existing matching flow. Triggered orders use the risk decision made at placement time; v3.2.0 does not add a second activation-time pre-trade risk call.

## Idempotency And Concurrency

The worker scans eligible stop orders with PostgreSQL row locks and `SKIP LOCKED`, then conditionally updates rows where `triggered_at IS NULL`. Multiple worker replicas can poll safely: the first committed trigger writes `triggered_at`, `original_order_type`, `activated_order_type`, and `stop_trigger_reference_price`; later polls skip the order.

The `order.triggered` event is inserted into `order_events` and `order_outbox` in the same database transaction as activation and matching. Kafka publication remains owned by the transactional outbox publisher.

## Configuration

- `STOP_TRIGGER_ENABLED=true`
- `STOP_TRIGGER_POLL_INTERVAL=5s`
- `STOP_TRIGGER_BATCH_SIZE=100`
- `STOP_TRIGGER_MAX_REFERENCE_PRICE_AGE=5m`
- `STOP_TRIGGER_PROCESSING_TIMEOUT=10s`
- `STOP_TRIGGER_SHUTDOWN_TIMEOUT=15s`
- `STOP_TRIGGER_MARKET_TOPIC=market.ticks`
- `STOP_TRIGGER_CONSUMER_GROUP=order-service-stop-trigger`

## Operations

Status endpoint:

```bash
curl http://localhost:8086/internal/stop-trigger/status
```

Metrics:

- `tradeops_stop_trigger_polls_total{status}`
- `tradeops_stop_trigger_orders_triggered_total{symbol,side,order_type}`
- `tradeops_stop_trigger_errors_total{reason}`
- `tradeops_stop_trigger_lag_seconds`
- `tradeops_reference_price_age_seconds{symbol}`

## Known Limitations

- Reference prices are latest-price snapshots, not exchange-specific NBBO or venue-aware marks.
- Activation uses the original placement-time risk decision.
- The first implementation consumes the existing market-data Kafka topic and accepts both `market.price.updated` and legacy tick event names.
