# Portfolio Trade Processing

As of v3.1.2, `trade.executed` is the single source of truth for Portfolio Service financial mutations.

Each execution is applied once per tenant by `tenant_id + execution_id`. The event ID is also unique, and a normalized payload hash is stored so a republished execution with different financial fields is treated as a data integrity conflict instead of a normal duplicate.

Processing happens in one PostgreSQL transaction: validate the event, check idempotency, create or lock buyer and seller portfolios, lock cash and holdings in deterministic order, apply buyer and seller mutations, insert two ledger rows, record the processed execution, and commit.

`order.filled` may still be consumed for compatibility, but it does not mutate cash or holdings. This prevents final filled-order events from double-applying trades that have already been applied incrementally through one or more executions.

Current limitations: fees are zero, only USD is supported, short selling and negative cash are disabled, margin and settlement timing are not modeled, trade busts/corrections are not implemented, and Kafka exactly-once delivery is not claimed.
