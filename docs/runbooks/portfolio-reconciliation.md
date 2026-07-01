# Portfolio Reconciliation Runbook

Use this runbook when a `trade.executed` event does not appear to match portfolio state.

1. Confirm the Order Service execution row exists in `order_executions` and note tenant, execution ID, buyer/seller IDs, symbol, quantity, price, and correlation ID.
2. Confirm the Kafka payload contains the same execution ID and ownership fields.
3. Check `portfolio_processed_events` for `tenant_id + execution_id`.
4. If a processed row exists and `payload_hash` differs from the current payload, inspect the Portfolio DLQ for a payload conflict.
5. Check `portfolio_transactions` for two rows with the same execution ID: one `BUY`, one `SELL`.
6. Verify buyer cash decreased and buyer holding increased.
7. Verify seller holding decreased, seller cash increased, and realized PnL matches average-cost accounting.
8. Check duplicate, failed, payload conflict, DLQ, and reconciliation failure metrics.

Rollout order: apply migration `010`, deploy Order Service so execution events include ownership fields, deploy Portfolio Service consuming `trade.executed`, verify processed and duplicate metrics, then keep `order.filled` mutation disabled.
