# Matching Engine

The order matching code lives in `services/order-service/internal/matching`.

It accepts one incoming order plus a tenant-scoped, symbol-scoped set of resting orders. The book sorts bids by descending limit price and asks by ascending limit price. Equal-price orders keep FIFO priority by accepted creation time.

Crossing rules:

- market buy matches best asks
- market sell matches best bids
- limit buy matches asks priced at or below the limit
- limit sell matches bids priced at or above the limit

Execution quantity is the lesser remaining quantity. Execution price is always the resting maker price. The engine returns a pure result; persistence, advisory locks, outbox rows, and Kafka publication stay outside the package.

Time in force:

- `GTC`, `DAY`, and valid `GTD` limit orders may rest
- `IOC` executes immediately and cancels any remainder
- `FOK` prechecks full executable quantity and has no partial side effects
- `MARKET` never rests
