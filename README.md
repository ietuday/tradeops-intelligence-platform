# TradeOps Intelligence Platform

TradeOps Intelligence Platform is an enterprise-grade local trading, risk, AI analytics, and observability platform.

The platform is designed to run completely on a local machine without any cloud account.

## Current Scope

The repository currently includes the platform foundation, identity/RBAC, market data streaming, order management, portfolio management, and strategy backtesting.

### Included

- Monorepo structure
- Node.js TypeScript API Gateway
- Go identity service with JWT, refresh tokens, RBAC, audit logs, PostgreSQL, Redis, and Prometheus metrics
- Go market data service with MQTT ingestion, tick validation, PostgreSQL persistence, Redpanda publishing, simulator, and Prometheus metrics
- Go order service with simulated order lifecycle, JWT validation, RBAC, idempotency keys, PostgreSQL persistence, Redpanda events, and Prometheus metrics
- Go portfolio service with filled-order consumption, holdings and cash updates, snapshots, realized P&L, JWT validation, Redpanda events, and Prometheus metrics
- Python FastAPI strategy service with strategy CRUD, backtesting, signal generation, PostgreSQL persistence, Redpanda events, JWT/RBAC, and Prometheus metrics
- Angular shell placeholder
- React trading dashboard placeholder
- PostgreSQL
- Redis
- Eclipse Mosquitto MQTT broker
- Redpanda Kafka-compatible event bus
- Prometheus
- Grafana
- Docker Compose setup
- Makefile
- Smoke test script
- Release notes

### Not Included Yet

- Risk engine
- Notification service
- AI assistant bot
- Local Kubernetes deployment

## Local Prerequisites

- Docker
- Docker Compose
- Make
- Node.js, optional for local non-Docker development

## Local URLs

| Component | URL |
|---|---|
| API Gateway | http://localhost:8080 |
| API Gateway Health | http://localhost:8080/health |
| API Gateway Ready | http://localhost:8080/ready |
| API Gateway Metrics | http://localhost:8080/metrics |
| Identity Service | http://localhost:8084 |
| Market Data Service | http://localhost:8085 |
| Order Service | http://localhost:8086 |
| Portfolio Service | http://localhost:8087 |
| Strategy Service | http://localhost:8088 |
| Angular Shell | http://localhost:4200 |
| React Trading Dashboard | http://localhost:4300 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| Mosquitto MQTT | localhost:1883 |
| Redpanda Kafka | localhost:9092 |

## Market Data Streaming

The market data service subscribes to MQTT topic `market/+/tick`, validates incoming ticks, stores accepted ticks in PostgreSQL, and publishes normalized events to Redpanda topic `market.ticks`.

Expected MQTT tick payload:

```json
{
  "symbol": "AAPL",
  "price": 184.52,
  "volume": 1200,
  "source": "local-simulator",
  "eventTime": "2026-05-30T12:00:00Z"
}
```

When `MARKET_SIMULATOR_ENABLED=true`, the service publishes local simulated ticks every `MARKET_SIMULATOR_INTERVAL_MS` milliseconds for `AAPL`, `TSLA`, `MSFT`, `BTC-USD`, `ETH-USD`, `NIFTY50`, and `BANKNIFTY`.

Gateway routes:

```text
GET /api/market/health
GET /api/market/ready
GET /api/market/metrics
GET /api/market/ticks/latest
GET /api/market/symbols
```

## Order Management

The order service supports simulated trading orders through the API Gateway. Health and metrics are public; order APIs require a JWT from the identity service and enforce role-based access.

Create a MARKET order:

```bash
TOKEN="<access token from /api/auth/login>"

curl -X POST http://localhost:8080/api/orders \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-order-1" \
  -d '{
    "symbol": "AAPL",
    "side": "BUY",
    "orderType": "MARKET",
    "quantity": 10,
    "limitPrice": null,
    "stopPrice": null
  }'
```

Order API routes:

```text
GET  /api/orders/health
GET  /api/orders/ready
GET  /api/orders/metrics
POST /api/orders
GET  /api/orders
GET  /api/orders/{id}
POST /api/orders/{id}/cancel
```

Order events are published to Redpanda topics `order.created`, `order.validated`, `order.accepted`, `order.filled`, `order.rejected`, and `order.cancelled`.

## Portfolio Management

The portfolio service consumes `order.filled` events from Redpanda, updates user cash and holdings transactionally in PostgreSQL, stores snapshots, and publishes `portfolio.updated` and `portfolio.snapshot.created` events.

Portfolio API routes:

```text
GET /api/portfolio/health
GET /api/portfolio/ready
GET /api/portfolio/metrics
GET /api/portfolio
GET /api/portfolio/holdings
GET /api/portfolio/snapshots
GET /api/portfolio/exposure
GET /api/portfolio/pnl
```

Example:

```bash
TOKEN="<access token from /api/auth/login>"

curl http://localhost:8080/api/portfolio/holdings \
  -H "Authorization: Bearer ${TOKEN}"
```

The local flow is:

```text
POST /api/orders -> order.filled -> portfolio-service -> portfolio.updated
```

## Strategy Studio And Backtesting

The strategy service reads market ticks from PostgreSQL, runs backtests for supported strategies, stores performance and generated signals, and publishes strategy events to Redpanda.

Supported strategy types:

```text
MOVING_AVERAGE_CROSSOVER
RSI
VOLATILITY_BREAKOUT
```

Strategy API routes:

```text
GET  /api/strategies/health
GET  /api/strategies/ready
GET  /api/strategies/metrics
POST /api/strategies
GET  /api/strategies
GET  /api/strategies/{id}
POST /api/strategies/{id}/backtest
GET  /api/strategies/{id}/performance
GET  /api/strategies/{id}/signals
```

Example:

```bash
TOKEN="<access token from /api/auth/login>"

curl -X POST http://localhost:8080/api/strategies \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AAPL MA Cross",
    "symbol": "AAPL",
    "strategyType": "MOVING_AVERAGE_CROSSOVER",
    "parameters": {
      "shortWindow": 5,
      "longWindow": 20
    }
  }'

curl -X POST http://localhost:8080/api/strategies/<strategy-id>/backtest \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "startTime": "2026-06-02T00:00:00Z",
    "endTime": "2026-06-02T01:00:00Z",
    "initialCapital": 100000
  }'
```

The local flow is:

```text
market-data-service -> market_ticks -> strategy-service backtest -> strategy.signal.generated
```

## Local Docker Environment

Create a local Docker environment file before starting the stack:

```bash
cp infrastructure/docker/.env.example infrastructure/docker/.env
```

Edit `infrastructure/docker/.env` and set local-only values for:

```bash
POSTGRES_PASSWORD=
GRAFANA_ADMIN_PASSWORD=
IDENTITY_DATABASE_URL=
IDENTITY_JWT_SECRET=
IDENTITY_REFRESH_TOKEN_SECRET=
MARKET_DATA_DATABASE_URL=
ORDER_DATABASE_URL=
ORDER_JWT_SECRET=
PORTFOLIO_DATABASE_URL=
PORTFOLIO_JWT_SECRET=
STRATEGY_DATABASE_URL=
STRATEGY_JWT_SECRET=
```

For local Compose, both database URLs can point at the local Postgres service, for example:

```bash
IDENTITY_DATABASE_URL=postgres://tradeops:<password>@postgres:5432/tradeops?sslmode=disable
MARKET_DATA_DATABASE_URL=postgres://tradeops:<password>@postgres:5432/tradeops?sslmode=disable
ORDER_DATABASE_URL=postgres://tradeops:<password>@postgres:5432/tradeops?sslmode=disable
ORDER_JWT_SECRET=<same value as IDENTITY_JWT_SECRET>
PORTFOLIO_DATABASE_URL=postgres://tradeops:<password>@postgres:5432/tradeops?sslmode=disable
PORTFOLIO_JWT_SECRET=<same value as IDENTITY_JWT_SECRET>
STRATEGY_DATABASE_URL=postgresql+psycopg://tradeops:<password>@postgres:5432/tradeops
STRATEGY_JWT_SECRET=<same value as IDENTITY_JWT_SECRET>
```

Do not commit `infrastructure/docker/.env`; it is intentionally ignored because it contains local secrets.

## Commands

```bash
make dev-up
make ps
make smoke
make logs
make dev-down
```
