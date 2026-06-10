# API Summary

All client-facing APIs are available through the API Gateway at `http://localhost:8080`. Direct service ports remain available for health checks and local debugging.

Most business APIs require a JWT access token from `/api/auth/login`.

```bash
TOKEN="<access token from /api/auth/login>"
```

## Tenant Context

Tenant-owned APIs use the JWT `tenantId` claim. API Gateway forwards the resolved tenant to services as `X-Tenant-ID`. External `X-Tenant-ID` cannot override a JWT tenant unless the caller has `trading_admin`; local demos default to `default-tenant`.

Tenant-owned resources such as orders, portfolios, surveillance alerts, notifications, and audit logs are filtered by tenant where implemented. Market data remains global by default.

## Auth APIs

Base gateway path: `/api/auth`

Auth required: No for register/login/refresh; token-oriented endpoints depend on identity-service behavior.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/auth/health` | Identity service health. |
| `GET` | `/api/auth/ready` | Identity readiness. |
| `POST` | `/api/auth/register` | Register a user. |
| `POST` | `/api/auth/login` | Login and receive tokens. |
| `POST` | `/api/auth/refresh` | Refresh access token. |
| `POST` | `/api/auth/logout` | Logout/revoke refresh state. |

Example:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  --data '{"email":"demo.trader@example.com","password":"Password@123"}'
```

## Market APIs

Base gateway path: `/api/market`

Auth required: Health/ready/metrics no; market data endpoints may be used for local demo reads.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/market/health` | Market data service health. |
| `GET` | `/api/market/ready` | Market data readiness. |
| `GET` | `/api/market/metrics` | Market metrics. |
| `GET` | `/api/market/ticks/latest` | Latest normalized market ticks. |
| `GET` | `/api/market/symbols` | Known market symbols. |

Example:

```bash
curl http://localhost:8080/api/market/ticks/latest
```

## Order APIs

Base gateway path: `/api/orders`

Auth required: Yes for order workflow APIs.

Roles: trading user roles are expected; local demo tokens commonly use `trader`.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/orders/health` | Order service health. |
| `GET` | `/api/orders/ready` | Order readiness. |
| `GET` | `/api/orders/metrics` | Order metrics. |
| `POST` | `/api/orders` | Create an order. |
| `GET` | `/api/orders` | List orders. |
| `GET` | `/api/orders/{id}` | Get an order. |
| `POST` | `/api/orders/{id}/cancel` | Cancel an order. |

Example:

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-order-1" \
  --data '{"symbol":"AAPL","side":"BUY","orderType":"MARKET","quantity":10,"limitPrice":null,"stopPrice":null}'
```

## Portfolio APIs

Base gateway path: `/api/portfolio`

Auth required: Yes for portfolio data APIs.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/portfolio/health` | Portfolio service health. |
| `GET` | `/api/portfolio/ready` | Portfolio readiness. |
| `GET` | `/api/portfolio/metrics` | Portfolio metrics. |
| `GET` | `/api/portfolio/holdings` | Current holdings. |
| `GET` | `/api/portfolio/summary` | Portfolio summary when available. |

Example:

```bash
curl http://localhost:8080/api/portfolio/holdings \
  -H "Authorization: Bearer ${TOKEN}"
```

## Strategy APIs

Base gateway path: `/api/strategies`

Auth required: Yes for strategy workflow APIs.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/strategies/health` | Strategy service health. |
| `GET` | `/api/strategies/ready` | Strategy readiness. |
| `GET` | `/api/strategies/metrics` | Strategy metrics. |
| `POST` | `/api/strategies` | Create a strategy. |
| `GET` | `/api/strategies` | List strategies. |
| `GET` | `/api/strategies/{id}` | Get a strategy. |
| `POST` | `/api/strategies/{id}/backtest` | Run a backtest. |
| `GET` | `/api/strategies/{id}/performance` | Strategy performance. |
| `GET` | `/api/strategies/{id}/signals` | Generated signals. |

Example:

```bash
curl -X POST http://localhost:8080/api/strategies \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"name":"Demo MA Cross","symbol":"AAPL","strategyType":"MOVING_AVERAGE_CROSSOVER","parameters":{"shortWindow":1,"longWindow":2}}'
```

## Risk APIs

Base gateway path: `/api/risk`

Auth required: Yes for portfolio risk data APIs.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/risk/health` | Risk engine health. |
| `GET` | `/api/risk/ready` | Risk engine readiness. |
| `GET` | `/api/risk/metrics` | Risk metrics. |
| `GET` | `/api/risk/portfolio/score` | Current risk score. |
| `GET` | `/api/risk/portfolio/volatility` | Volatility calculation. |
| `GET` | `/api/risk/portfolio/drawdown` | Drawdown calculation. |
| `GET` | `/api/risk/portfolio/var` | Value at Risk calculation. |
| `GET` | `/api/risk/recommendations` | Risk recommendations. |
| `GET` | `/api/risk/anomalies` | Risk anomalies. |

Example:

```bash
curl http://localhost:8080/api/risk/portfolio/score \
  -H "Authorization: Bearer ${TOKEN}"
```

## Surveillance APIs

Base gateway path: `/api/surveillance`

Auth required: Yes for alert APIs.

Roles: risk/surveillance-oriented roles are expected; local demo tokens commonly use `risk_manager`.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/surveillance/health` | Surveillance health. |
| `GET` | `/api/surveillance/ready` | Surveillance readiness. |
| `GET` | `/api/surveillance/metrics` | Surveillance metrics. |
| `GET` | `/api/surveillance/alerts` | List alerts. |
| `GET` | `/api/surveillance/alerts/summary` | Alert summary. |
| `GET` | `/api/surveillance/alerts/{id}` | Get alert detail. |
| `POST` | `/api/surveillance/alerts/{id}/acknowledge` | Move alert to `ACKNOWLEDGED`. |
| `POST` | `/api/surveillance/alerts/{id}/resolve` | Move alert to `RESOLVED`. |
| `POST` | `/api/surveillance/alerts/{id}/dismiss` | Move alert to `DISMISSED`. |
| `GET` | `/api/surveillance/rules` | List tenant-effective rule configs. |
| `GET` | `/api/surveillance/rules/{ruleName}` | Get one rule config. |
| `PUT` | `/api/surveillance/rules/{ruleName}` | Update threshold/severity/enabled state. |
| `POST` | `/api/surveillance/rules/{ruleName}/enable` | Enable a rule. |
| `POST` | `/api/surveillance/rules/{ruleName}/disable` | Disable a rule. |

Example:

```bash
curl "http://localhost:8080/api/surveillance/alerts?status=OPEN&limit=10" \
  -H "Authorization: Bearer ${TOKEN}"
```

Rule config writes require `trading_admin` or `risk_manager`:

```bash
curl -X PUT "http://localhost:8080/api/surveillance/rules/LargeOrderRule" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"enabled":true,"severity":"CRITICAL","thresholdNumeric":250000}'
```

## Notification APIs

Base gateway path: `/api/notifications`

Auth required: Yes for notification and preference APIs.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/notifications/health` | Notification service health. |
| `GET` | `/api/notifications/ready` | Notification readiness. |
| `GET` | `/api/notifications/metrics` | Notification metrics. |
| `GET` | `/api/notifications` | List notifications. |
| `GET` | `/api/notifications/summary` | Notification summary. |
| `GET` | `/api/notifications/preferences` | Get notification preferences. |
| `PUT` | `/api/notifications/preferences` | Update notification preferences. |
| `GET` | `/api/notifications/{id}` | Get notification detail. |
| `POST` | `/api/notifications/{id}/mark-read` | Move notification to `READ`. |
| `POST` | `/api/notifications/{id}/retry` | Request retry for failed delivery. |

Example:

```bash
curl "http://localhost:8080/api/notifications?limit=20" \
  -H "Authorization: Bearer ${TOKEN}"
```

## Audit APIs

Base gateway path: `/api/audit`

Auth required: Yes for audit log, summary, and export APIs.

Roles: read access allows `trading_admin`, `risk_manager`, and `analyst`; export allows `trading_admin` and `risk_manager`.

Common endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/audit/health` | Audit service health. |
| `GET` | `/api/audit/ready` | Audit readiness. |
| `GET` | `/api/audit/metrics` | Audit metrics. |
| `GET` | `/api/audit/logs` | List audit logs with filters. |
| `GET` | `/api/audit/logs/{id}` | Get audit log detail. |
| `GET` | `/api/audit/summary` | Audit counts by service, event type, severity, and action. |
| `GET` | `/api/audit/export?format=json` | Export filtered audit logs as JSON. |
| `GET` | `/api/audit/export?format=csv` | Export filtered audit logs as CSV. |

Example:

```bash
curl "http://localhost:8080/api/audit/logs?serviceName=order-service&limit=20" \
  -H "Authorization: Bearer ${TOKEN}"
```
