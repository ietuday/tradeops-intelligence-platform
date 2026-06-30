# Transactional Outbox

New OMS state changes write `order_events` and `order_outbox` in the same database transaction as the order and execution updates.

The outbox schema stores event id, type, aggregate id, topic, payload, correlation id, trace metadata, status, attempt count, retry availability, publication timestamp, and last error.

Delivery semantics are at-least-once. Publishers must only mark rows published after Kafka accepts the message. Consumers must deduplicate by event id or execution id.
