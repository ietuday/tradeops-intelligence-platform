# TradeOps Intelligence Platform

TradeOps Intelligence Platform is an enterprise-grade local trading, risk, AI analytics, and observability platform.

The platform is designed to run completely on a local machine without any cloud account.

Latest release: `v1.0.0` Production Readiness & Platform Hardening.

## v1.0.0 Readiness Docs

- [Architecture overview](docs/architecture/overview.md)
- [Event-flow reference](docs/architecture/event-flow.md)
- [Service dependency matrix](docs/architecture/service-dependency-matrix.md)
- [API summary](docs/api/api-summary.md)
- [Production-readiness checklist](docs/production-readiness/checklist.md)
- [Troubleshooting guide](docs/troubleshooting.md)
- [Interview walkthrough](docs/interview/project-walkthrough.md)
- [v1.0.0 release notes](docs/release-notes/v1.0.0.md)
- [End-to-end demo script](scripts/demo-e2e-tradeops.sh)

## Current Scope

The repository currently includes the platform foundation, identity/RBAC, market data streaming, order management, portfolio management, strategy backtesting, portfolio risk analytics, trade surveillance alerting, and notification processing.

### Included

- Monorepo structure
- Node.js TypeScript API Gateway
- Go identity service with JWT, refresh tokens, RBAC, audit logs, PostgreSQL, Redis, and Prometheus metrics
- Go market data service with MQTT ingestion, tick validation, PostgreSQL persistence, Redpanda publishing, simulator, and Prometheus metrics
- Go order service with simulated order lifecycle, JWT validation, RBAC, idempotency keys, PostgreSQL persistence, Redpanda events, and Prometheus metrics
- Go portfolio service with filled-order consumption, holdings and cash updates, snapshots, realized P&L, JWT validation, Redpanda events, and Prometheus metrics
- Python FastAPI strategy service with strategy CRUD, backtesting, signal generation, PostgreSQL persistence, Redpanda events, JWT/RBAC, and Prometheus metrics
- Python FastAPI risk engine service with portfolio risk score, volatility, drawdown, VaR, anomalies, recommendations, PostgreSQL persistence, Redpanda events, JWT/RBAC, and Prometheus metrics
- Go surveillance service with rule-based alerting, PostgreSQL persistence, Redpanda consumption/publishing, JWT/RBAC, and Prometheus metrics
- Go notification service with notification persistence, preferences, surveillance alert event consumption, mock email, webhook delivery, Redpanda publishing, JWT/RBAC, and Prometheus metrics
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
| Risk Engine Service | http://localhost:8089 |
| Surveillance Service | http://localhost:8090 |
| Notification Service | http://localhost:8091 |
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

## Risk Engine AI/ML

The risk engine service reads portfolio, snapshot, holdings, market tick, order, and strategy-derived data from PostgreSQL, calculates portfolio risk analytics, persists outputs, and publishes risk events to Redpanda.

Risk API routes:

```text
GET /api/risk/health
GET /api/risk/ready
GET /api/risk/metrics
GET /api/risk/portfolio/score
GET /api/risk/portfolio/volatility
GET /api/risk/portfolio/drawdown
GET /api/risk/portfolio/var
GET /api/risk/recommendations
GET /api/risk/anomalies
```

Example:

```bash
TOKEN="<access token from /api/auth/login>"

curl http://localhost:8080/api/risk/portfolio/score \
  -H "Authorization: Bearer ${TOKEN}"

curl http://localhost:8080/api/risk/recommendations \
  -H "Authorization: Bearer ${TOKEN}"
```

Risk events are published to Redpanda topics `risk.score.updated`, `risk.breached`, `risk.anomaly.detected`, and `risk.recommendation.created`.

The local flow is:

```text
portfolio.updated + portfolio.snapshot.created + market.ticks -> risk-engine-service -> risk.score.updated
```

## Trade Surveillance And Alerting

The surveillance service consumes trading, portfolio, risk, market, and strategy events from Redpanda, evaluates simple in-code surveillance rules, stores alerts in PostgreSQL, and publishes alert lifecycle events.

Initial rules:

```text
LargeOrderRule
RapidOrderSubmissionRule
HighCancelRateRule
RiskScoreBreachRule
AbnormalPriceMovementRule
```

Surveillance API routes:

```text
GET  /api/surveillance/health
GET  /api/surveillance/ready
GET  /api/surveillance/metrics
GET  /api/surveillance/alerts
GET  /api/surveillance/alerts/summary
GET  /api/surveillance/alerts/{id}
POST /api/surveillance/alerts/{id}/acknowledge
POST /api/surveillance/alerts/{id}/resolve
POST /api/surveillance/alerts/{id}/dismiss
```

Example:

```bash
TOKEN="<access token from /api/auth/login>"

curl "http://localhost:8080/api/surveillance/alerts?status=OPEN&limit=50" \
  -H "Authorization: Bearer ${TOKEN}"

curl -X POST http://localhost:8080/api/surveillance/alerts/<alert-id>/acknowledge \
  -H "Authorization: Bearer ${TOKEN}"
```

Surveillance consumes Redpanda topics `order.created`, `order.filled`, `order.cancelled`, `portfolio.updated`, `risk.score.updated`, `market.ticks`, and `strategy.signal.generated`.

Surveillance events are published to Redpanda topics `surveillance.alert.created`, `surveillance.alert.acknowledged`, `surveillance.alert.resolved`, and `surveillance.alert.dismissed`.

## Demo: Trade Surveillance & Alerting

Start the local Docker Compose stack:

```bash
make dev-up
```

Verify the surveillance service directly and through the API Gateway:

```bash
curl http://localhost:8090/health
curl http://localhost:8080/api/surveillance/health
```

Run the guided demo:

```bash
./scripts/demo-surveillance.sh
```

The script checks service health, publishes the sample large-order event when Redpanda is available through Docker Compose, lists alerts through the API Gateway, then moves one alert from `OPEN` to `ACKNOWLEDGED` to `RESOLVED`.

To trigger `LargeOrderRule` manually:

```bash
node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync("docs/examples/surveillance/order-created-large-order.json", "utf8"))) + "\n");' | \
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce order.created
```

List open alerts with a token that has `risk_manager` or `trading_admin` role:

```bash
TOKEN="<risk manager or trading admin access token>"

curl "http://localhost:8080/api/surveillance/alerts?status=OPEN&limit=50" \
  -H "Authorization: Bearer ${TOKEN}"
```

Acknowledge and resolve an alert:

```bash
ALERT_ID="<alert id>"

curl -X POST "http://localhost:8080/api/surveillance/alerts/${ALERT_ID}/acknowledge" \
  -H "Authorization: Bearer ${TOKEN}"

curl -X POST "http://localhost:8080/api/surveillance/alerts/${ALERT_ID}/resolve" \
  -H "Authorization: Bearer ${TOKEN}"
```

Prometheus metrics to check:

```text
surveillance_alerts_created_total
surveillance_alerts_acknowledged_total
surveillance_alerts_resolved_total
surveillance_rule_matches_total
surveillance_rule_executions_total
surveillance_kafka_messages_total
surveillance_kafka_publish_errors_total
surveillance_rule_duration_seconds
```

## Notification Service

The notification service consumes surveillance alert lifecycle events from Redpanda, creates user notifications, records delivery attempts, supports notification preferences, and publishes notification lifecycle events.

Notification API routes:

```text
GET  /api/notifications/health
GET  /api/notifications/ready
GET  /api/notifications/metrics
GET  /api/notifications
GET  /api/notifications/summary
GET  /api/notifications/preferences
PUT  /api/notifications/preferences
GET  /api/notifications/{id}
POST /api/notifications/{id}/mark-read
POST /api/notifications/{id}/retry
```

Example:

```bash
TOKEN="<access token from /api/auth/login>"

curl "http://localhost:8080/api/notifications?limit=50" \
  -H "Authorization: Bearer ${TOKEN}"

curl -X POST "http://localhost:8080/api/notifications/<notification-id>/mark-read" \
  -H "Authorization: Bearer ${TOKEN}"
```

Notification consumes Redpanda topics `surveillance.alert.created`, `surveillance.alert.acknowledged`, `surveillance.alert.resolved`, and `surveillance.alert.dismissed`.

Notification events are published to Redpanda topics `notification.created`, `notification.sent`, `notification.failed`, `notification.read`, and `notification.retry_requested`.

Prometheus metrics to check:

```text
notification_events_processed_total
notification_events_failed_total
notifications_created_total
notifications_marked_read_total
notification_retries_total
notification_preferences_updated_total
notification_delivery_attempts_total
notification_delivery_failures_total
notification_status_updates_total
notification_delivery_duration_seconds
```

## Demo: Notification Service

Start the local Docker Compose stack:

```bash
make dev-up
```

Verify the notification service directly and through the API Gateway:

```bash
curl http://localhost:8091/health
curl http://localhost:8080/api/notifications/health
curl http://localhost:8080/api/notifications/ready
```

Run the guided demo:

```bash
./scripts/demo-notifications.sh
```

The script checks service health, publishes a sample `surveillance.alert.created` event when Redpanda is available through Docker Compose, lists notifications through the API Gateway, and marks one notification as `READ`.

To publish the sample surveillance alert event manually:

```bash
node -e 'const fs = require("fs"); process.stdout.write(JSON.stringify(JSON.parse(fs.readFileSync("docs/examples/notifications/surveillance-alert-created-high.json", "utf8"))) + "\n");' | \
  docker compose -f infrastructure/docker/docker-compose.yml exec -T redpanda rpk topic produce surveillance.alert.created
```

List notifications with a user token:

```bash
TOKEN="<access token from /api/auth/login>"

curl "http://localhost:8080/api/notifications?limit=20" \
  -H "Authorization: Bearer ${TOKEN}"
```

Mark a notification as read:

```bash
curl -X POST "http://localhost:8080/api/notifications/<notification-id>/mark-read" \
  -H "Authorization: Bearer ${TOKEN}"
```

Configure a webhook preference:

```bash
curl -X PUT "http://localhost:8080/api/notifications/preferences" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data @docs/examples/notifications/notification-preference-webhook.json
```

Check notification metrics:

```bash
curl http://localhost:8091/metrics | grep '^notification_'
curl http://localhost:8080/api/notifications/metrics | grep '^notification_'
```

## Demo: End-To-End TradeOps Flow

Run the guided platform demo after Docker Compose is healthy:

```bash
./scripts/demo-e2e-tradeops.sh
```

The script checks service health, uses a local demo JWT or explains the token requirement, publishes the large-order sample event when Redpanda is available, waits for a surveillance alert, acknowledges it, lists notifications, and marks one notification as `READ`.

## Notification Troubleshooting

`notification-service` is not healthy:
Check `docker compose -f infrastructure/docker/docker-compose.yml ps notification-service`, then inspect logs with `docker compose -f infrastructure/docker/docker-compose.yml logs notification-service`. Confirm `NOTIFICATION_DATABASE_URL`, `NOTIFICATION_JWT_SECRET`, Postgres, and Redpanda are available.

Redpanda topic is not receiving events:
Verify Redpanda is running and publish the sample payload manually with `rpk topic produce surveillance.alert.created`. Confirm the notification service logs show the consumer started.

Webhook delivery failed:
Confirm the saved webhook URL is reachable from inside the Docker network, returns a 2xx response, and does not exceed the sender timeout. Failed attempts are recorded in `notification_delivery_attempts`.

Notification was not created:
Use `docs/examples/notifications/surveillance-alert-created-high.json` as a known-good payload. Check `notification_events_failed_total`, service logs, and the `notifications` table for the target user.

JWT/RBAC failure:
Use a fresh token from `/api/auth/login` or set `NOTIFICATION_DEMO_TOKEN` before running `scripts/demo-notifications.sh`. Local services should share the same JWT secret as identity.

Database migration missing:
Restart `notification-service` after Postgres is healthy. The service runs its migration on startup; if the tables are still absent, check the database URL and startup logs for migration errors.

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
RISK_DATABASE_URL=
RISK_JWT_SECRET=
SURVEILLANCE_DATABASE_URL=
SURVEILLANCE_JWT_SECRET=
NOTIFICATION_DATABASE_URL=
NOTIFICATION_JWT_SECRET=
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
RISK_DATABASE_URL=postgresql+psycopg://tradeops:<password>@postgres:5432/tradeops
RISK_JWT_SECRET=<same value as IDENTITY_JWT_SECRET>
SURVEILLANCE_DATABASE_URL=postgres://tradeops:<password>@postgres:5432/tradeops?sslmode=disable
SURVEILLANCE_JWT_SECRET=<same value as IDENTITY_JWT_SECRET>
NOTIFICATION_DATABASE_URL=postgres://tradeops:<password>@postgres:5432/tradeops?sslmode=disable
NOTIFICATION_JWT_SECRET=<same value as IDENTITY_JWT_SECRET>
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
