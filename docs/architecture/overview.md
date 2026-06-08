# TradeOps Architecture Overview

TradeOps Intelligence Platform is a local, enterprise-style trading intelligence system built from small services. It demonstrates authentication, market data ingestion, order flow, portfolio updates, strategy/risk analytics, surveillance alerts, notifications, audit trails, observability, and Docker Compose operations.

The platform is designed for portfolio and interview demonstration. It is not yet a real production deployment, but the service boundaries, messaging patterns, health checks, metrics, and runbooks mirror production concerns.

## Service List

| Service | Purpose |
| --- | --- |
| `api-gateway` | Single external HTTP entry point and reverse proxy to backend services. |
| `identity-service` | Registration, login, JWT issuance, refresh tokens, and RBAC identity data. |
| `market-data-service` | MQTT market tick ingestion, validation, storage, and Kafka publication. |
| `order-service` | Order creation, validation, idempotency, status transitions, and order events. |
| `portfolio-service` | Consumes fills, updates holdings/cash, and publishes portfolio snapshots. |
| `strategy-service` | Strategy CRUD, backtests, performance, generated signals, and strategy events. |
| `risk-engine-service` | Portfolio risk score, VaR, volatility, drawdown, recommendations, and risk events. |
| `surveillance-service` | Consumes trading/risk/market events and creates alert lifecycle events. |
| `notification-service` | Consumes surveillance alert events, creates notifications, and records delivery attempts. |
| `audit-service` | Consumes platform events, stores searchable audit logs, and exposes summary/export APIs. |

## Technology Stack

| Layer | Technology |
| --- | --- |
| Gateway | Node.js, Express, Jest |
| Go services | Go, Chi, pgx, kafka-go, Prometheus client |
| Python services | FastAPI, SQLAlchemy, psycopg, confluent-kafka, Prometheus client |
| Data | PostgreSQL, Redis |
| Messaging | Redpanda/Kafka, Mosquitto/MQTT |
| Observability | Prometheus, Grafana, alert rules, SLO docs, structured logs, correlation IDs |
| Runtime | Docker Compose, optional Helm/Kubernetes deployment-readiness chart |

## High-Level Architecture

```mermaid
flowchart LR
  user[User / Client] --> gateway[API Gateway<br/>:8080]

  gateway --> identity[Identity Service<br/>:8084]
  gateway --> market[Market Data Service<br/>:8085]
  gateway --> orders[Order Service<br/>:8086]
  gateway --> portfolio[Portfolio Service<br/>:8087]
  gateway --> strategy[Strategy Service<br/>:8088]
  gateway --> risk[Risk Engine Service<br/>:8089]
  gateway --> surveillance[Surveillance Service<br/>:8090]
  gateway --> notification[Notification Service<br/>:8091]
  gateway --> audit[Audit Service<br/>:8092]

  identity --> postgres[(PostgreSQL)]
  identity --> redis[(Redis)]
  market --> postgres
  orders --> postgres
  portfolio --> postgres
  strategy --> postgres
  risk --> postgres
  surveillance --> postgres
  notification --> postgres
  audit --> postgres

  mosquitto[(Mosquitto / MQTT)] --> market
  market --> redpanda[(Redpanda / Kafka)]
  orders --> redpanda
  portfolio --> redpanda
  strategy --> redpanda
  risk --> redpanda
  surveillance --> redpanda
  notification --> redpanda
  audit --> redpanda

  redpanda --> portfolio
  redpanda --> surveillance
  redpanda --> notification
  redpanda --> audit

  prometheus[Prometheus] --> gateway
  prometheus --> identity
  prometheus --> market
  prometheus --> orders
  prometheus --> portfolio
  prometheus --> strategy
  prometheus --> risk
  prometheus --> surveillance
  prometheus --> notification
  prometheus --> audit
  grafana[Grafana] --> prometheus
```

## Request Flow

1. A client sends HTTP requests to the API Gateway.
2. Public auth endpoints are proxied to `identity-service`.
3. Protected service APIs require a JWT issued by identity.
4. The gateway forwards the authorization and correlation headers to the target service.
5. Services validate JWT/RBAC locally, use PostgreSQL for service data, and return JSON responses.
6. Long-running side effects are represented as Kafka events when applicable.

## Event-Driven Flow

```mermaid
flowchart LR
  mqtt[MQTT market/+/tick] --> market[market-data-service]
  market --> t_market[market.ticks]

  orders[order-service] --> t_created[order.created]
  orders --> t_validated[order.validated]
  orders --> t_accepted[order.accepted]
  orders --> t_filled[order.filled]
  orders --> t_rejected[order.rejected]
  orders --> t_cancelled[order.cancelled]

  t_filled --> portfolio[portfolio-service]
  portfolio --> t_portfolio[portfolio.updated]
  portfolio --> t_snapshot[portfolio.snapshot.created]

  strategy[strategy-service] --> t_signal[strategy.signal.generated]
  strategy --> t_backtest[strategy.backtest.completed]

  risk[risk-engine-service] --> t_risk_score[risk.score.updated]
  risk --> t_risk_breached[risk.breached]
  risk --> t_risk_anomaly[risk.anomaly.detected]

  t_market --> surveillance[surveillance-service]
  t_created --> surveillance
  t_filled --> surveillance
  t_cancelled --> surveillance
  t_portfolio --> surveillance
  t_signal --> surveillance
  t_risk_score --> surveillance

  surveillance --> t_alert_created[surveillance.alert.created]
  surveillance --> t_alert_ack[surveillance.alert.acknowledged]
  surveillance --> t_alert_resolved[surveillance.alert.resolved]
  surveillance --> t_alert_dismissed[surveillance.alert.dismissed]

  t_alert_created --> notification[notification-service]
  t_alert_ack --> notification
  t_alert_resolved --> notification
  t_alert_dismissed --> notification

  notification --> t_notification_created[notification.created]
  notification --> t_notification_sent[notification.sent]
  notification --> t_notification_failed[notification.failed]
  notification --> t_notification_read[notification.read]
```

## Observability Flow

- Each service exposes `/health`, `/ready`, and `/metrics`.
- The API Gateway also proxies major service health, readiness, and metrics routes where supported.
- Prometheus scrapes gateway and service metrics over the Docker Compose network.
- Grafana reads Prometheus and includes dashboards for platform overview, API Gateway, event processing, surveillance/notifications, and audit/compliance.
- Prometheus loads local alert rules for service availability, gateway failures/latency, event processing failures, DLQ events, notification delivery failures, and audit ingestion failures.
- SLO-oriented docs live under `docs/observability/` and describe demo SLIs, dashboard usage, alert behavior, and runbook steps.
- Correlation IDs flow through the gateway to help trace requests across services.

## Deployment Readiness

- Docker Compose remains the primary local runtime and demo path.
- The optional Helm chart under `infrastructure/helm/tradeops-platform/` renders Kubernetes Deployments and Services for application services.
- Kubernetes config separates non-secret values in a ConfigMap from placeholder secret values in a Secret.
- Application pods include `/health` liveness probes, `/ready` readiness probes, resource requests/limits, and `terminationGracePeriodSeconds: 30`.
- Stateful infrastructure such as PostgreSQL, Redis, Redpanda, Mosquitto, Prometheus, and Grafana is expected to be managed separately for Kubernetes deployments.

## Security Flow

- Users register and login through `/api/auth`.
- `identity-service` issues JWT access tokens.
- Services verify JWT signatures using the shared local identity secret configured in Compose.
- Services enforce role-based access where implemented.
- Secrets are provided through `infrastructure/docker/.env` and should not be committed.

## Database Usage

- PostgreSQL is the primary store for identity, market, order, portfolio, strategy, risk, surveillance, and notification data.
- `audit-service` stores normalized audit logs and export request records in PostgreSQL.
- Services own their tables and run their own startup migrations.
- Redis is used by identity for refresh-token/session-oriented state.
- The current local Compose deployment uses one PostgreSQL database for convenience; stronger isolation would be expected in a production deployment.

## Messaging Usage

- Mosquitto receives raw/simulated market ticks.
- `market-data-service` normalizes MQTT ticks and publishes `market.ticks`.
- Redpanda/Kafka connects order, portfolio, strategy, risk, surveillance, and notification flows.
- Bad payload handling is intentionally defensive in event consumers so malformed demo messages do not crash services.
