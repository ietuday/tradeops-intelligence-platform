# Smart Order Execution

TradeOps v3.1.0 extends the Order Service from simulated fills to a deterministic OMS workflow.

## Implemented flow

1. The HTTP handler authenticates the user, enforces idempotency on create, and delegates to the service layer.
2. The service normalizes order input, defaults `timeInForce` to `DAY`, accepts `STOP_LOSS` as a `STOP` alias, and validates lifecycle fields.
3. The repository owns the placement transaction. It inserts the incoming order, acquires a transaction-scoped PostgreSQL advisory lock for `tenant:symbol`, loads eligible resting limit orders with `FOR UPDATE`, runs the matching engine, inserts immutable executions, updates affected order versions, writes `order_events`, writes `order_outbox`, and commits.
4. Kafka publication remains at-least-once. New OMS mutations also write an outbox row in the same transaction so an outbox publisher can retry safely.

## Matching semantics

The matching engine is deterministic and price-time-priority based:

- highest bid and lowest ask have best price priority
- equal prices use FIFO by `created_at`
- execution price is the resting maker price
- partial fills update cumulative filled and remaining quantity
- `MARKET` and `IOC` orders never rest unfilled remainder
- `FOK` prechecks available liquidity and executes nothing unless the order can fully fill

Money calculations in the matching package use exact rational arithmetic. Existing JSON compatibility is preserved with numeric API fields.

## Concurrency

The initial distributed ownership strategy is PostgreSQL advisory locking per `tenant:symbol`, held inside the placement transaction. Resting rows are selected `FOR UPDATE`, and order amendments use optimistic version checks. This is suitable for multiple Kubernetes replicas sharing PostgreSQL, but it is not a multi-region matching architecture.

## Known limitations

- This is a trading-system simulation, not an exchange-certified engine.
- No broker integration, FIX protocol, or real exchange calendar is included.
- `STOP` and `STOP_LIMIT` are validated and persisted but reference-price triggering is scaffolded for a later worker.
- Pre-trade risk decision storage is migrated; remote risk evaluation is not enabled in this slice.
- Outbox delivery is at-least-once, not exactly-once; consumers must be idempotent.
- DAY expiry worker configuration is documented but the worker is not enabled in this implementation slice.
