# Service Dependency Matrix

| Service | Language | Port | Database | Cache | Message Broker | Consumed Topics | Published Topics | External Dependencies | Health Endpoint | Metrics Endpoint |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `api-gateway` | Node.js / TypeScript | `8080` | None | None | None | None | None | Backend HTTP services | `/health`, `/ready` | `/metrics` |
| `identity-service` | Go | `8084` externally, `8080` in Compose | PostgreSQL | Redis | None | None | None | PostgreSQL, Redis | `/health`, `/ready`, `/api/auth/health`, `/api/auth/ready` | `/metrics`, `/api/auth/metrics` |
| `market-data-service` | Go | `8085` externally, `8080` in Compose | PostgreSQL | None | Mosquitto, Redpanda | MQTT `market/+/tick` | `market.ticks` | PostgreSQL, Mosquitto, Redpanda | `/health`, `/ready`, `/api/market/health`, `/api/market/ready` | `/metrics`, `/api/market/metrics` |
| `order-service` | Go | `8086` externally, `8080` in Compose | PostgreSQL | None | Redpanda | None | `order.created`, `order.validated`, `order.accepted`, `order.filled`, `order.rejected`, `order.cancelled` | PostgreSQL, Redpanda | `/health`, `/ready`, `/api/orders/health`, `/api/orders/ready` | `/metrics`, `/api/orders/metrics` |
| `portfolio-service` | Go | `8087` externally, `8080` in Compose | PostgreSQL | None | Redpanda | `order.filled` | `portfolio.updated`, `portfolio.snapshot.created` | PostgreSQL, Redpanda | `/health`, `/ready`, `/api/portfolio/health`, `/api/portfolio/ready` | `/metrics`, `/api/portfolio/metrics` |
| `strategy-service` | Python / FastAPI | `8088` externally, `8080` in Compose | PostgreSQL | None | Redpanda | None | `strategy.signal.generated`, `strategy.backtest.completed` | PostgreSQL, Redpanda | `/health`, `/ready`, `/api/strategies/health`, `/api/strategies/ready` | `/metrics`, `/api/strategies/metrics` |
| `risk-engine-service` | Python / FastAPI | `8089` externally, `8080` in Compose | PostgreSQL | None | Redpanda | None | `risk.score.updated`, `risk.breached`, `risk.anomaly.detected`, `risk.recommendation.created` | PostgreSQL, Redpanda | `/health`, `/ready`, `/api/risk/health`, `/api/risk/ready` | `/metrics`, `/api/risk/metrics` |
| `surveillance-service` | Go | `8090` | PostgreSQL | None | Redpanda | `order.created`, `order.filled`, `order.cancelled`, `portfolio.updated`, `risk.score.updated`, `market.ticks`, `strategy.signal.generated` | `surveillance.alert.created`, `surveillance.alert.acknowledged`, `surveillance.alert.resolved`, `surveillance.alert.dismissed` | PostgreSQL, Redpanda | `/health`, `/ready`, `/api/surveillance/health`, `/api/surveillance/ready` | `/metrics`, `/api/surveillance/metrics` |
| `notification-service` | Go | `8091` | PostgreSQL | None | Redpanda | `surveillance.alert.created`, `surveillance.alert.acknowledged`, `surveillance.alert.resolved`, `surveillance.alert.dismissed` | `notification.created`, `notification.sent`, `notification.failed`, `notification.read`, `notification.retry_requested` | PostgreSQL, Redpanda, optional webhook endpoints | `/health`, `/ready`, `/api/notifications/health`, `/api/notifications/ready` | `/metrics`, `/api/notifications/metrics` |

## Notes

- External ports are the host ports exposed by `infrastructure/docker/docker-compose.yml`.
- Most services listen on `8080` inside Docker; surveillance and notification intentionally use `8090` and `8091`.
- The API Gateway is the preferred entry point for client traffic.
- Direct service ports are useful for smoke tests and troubleshooting.
