# Order Lifecycle

Supported statuses:

- `created`
- `validated`
- `risk_pending`
- `risk_rejected`
- `accepted`
- `partially_filled`
- `filled`
- `cancelled`
- `rejected`
- `expired`

Domain transition helpers reject invalid jumps, such as `filled` to `cancelled` or `cancelled` to `accepted`. Accepted and partially filled orders may be cancelled. Amendments require an `expectedVersion` and return conflict on stale versions.

`STOP_LOSS` remains accepted at the API boundary and is normalized to `STOP`.
