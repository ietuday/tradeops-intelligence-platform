# Event Consumer Idempotency

| Consumer | Event Types | Idempotency Key | Persistence Mechanism | Duplicate Behavior | Gap / Follow-up |
| --- | --- | --- | --- | --- | --- |
| Portfolio Service | `trade.executed` | `tenantId:executionId`, with `eventId` also unique | `portfolio_processed_events` table in the same transaction as cash, holding, and ledger mutations | Duplicate executions are acknowledged without mutation; conflicting payload hashes are reconciliation failures | `order.filled` is lifecycle-only/no-op for financial state |
| Audit Service | order, portfolio, risk, surveillance, notification topics | `topic:eventId` when present | unique `audit_logs.source_event_key` index | Duplicate audit logs are skipped | Ensure all producers include stable `eventId` |
| Notification Service | surveillance alert events | source event ID + event type + channel | repository duplicate lookup before notification creation | Duplicate notification per channel is skipped | Keep source event metadata mandatory for future alert types |
| Surveillance Service | order, portfolio, risk, market, strategy topics | derived alert identity | duplicate alert repository check | Duplicate alerts are skipped | Rule execution records may still be written for duplicate source events |
