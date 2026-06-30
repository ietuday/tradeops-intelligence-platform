# Order Service Runbook

## Useful checks

- `GET /health`
- `GET /ready`
- `GET /metrics`
- `GET /orders`
- `GET /order-books/{symbol}/depth`
- `GET /trades`

## Operational notes

- Matching uses PostgreSQL advisory locks keyed by `tenant:symbol`.
- A growing `order_outbox` pending backlog means Kafka publication is failing or the outbox publisher is not running.
- Version conflicts on amendments are expected during concurrent user edits; clients should refetch the order and retry with the current version.
- `MARKET`, `IOC`, and failed `FOK` orders do not rest on the book.
