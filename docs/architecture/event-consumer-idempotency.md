# Event Consumer Idempotency

| Consumer | Event Types | Idempotency Key | Persistence Mechanism | Duplicate Behavior | Gap / Follow-up |
| --- | --- | --- | --- | --- | --- |
| Portfolio Service | `order.filled` | `eventId` | `processed_order_events` table keyed by tenant/event | Duplicate filled events are skipped | Migrate to consume `trade.executed` idempotently instead of relying on final `order.filled` events |
| Audit Service | order, portfolio, risk, surveillance, notification topics | `topic:eventId` when present | unique `audit_logs.source_event_key` index | Duplicate audit logs are skipped | Ensure all producers include stable `eventId` |
| Notification Service | surveillance alert events | source event ID + event type + channel | repository duplicate lookup before notification creation | Duplicate notification per channel is skipped | Keep source event metadata mandatory for future alert types |
| Surveillance Service | order, portfolio, risk, market, strategy topics | derived alert identity | duplicate alert repository check | Duplicate alerts are skipped | Rule execution records may still be written for duplicate source events |
