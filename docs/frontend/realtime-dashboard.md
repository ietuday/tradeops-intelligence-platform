# Real-Time Dashboard

v2.9.0 upgrades the existing React dashboard placeholder into a lightweight demo UI for the TradeOps backend. It uses the API Gateway, admin operations APIs, risk analytics APIs, and WebSocket streams.

Angular shell remains unchanged.

## Pages

| Page | Purpose |
| --- | --- |
| Dashboard | Platform status, alerts, notifications, DLQs, enabled rules, and service health. |
| Realtime | WebSocket stream selector, connection status, and last 50 events. |
| Risk Analytics | Built-in scenarios and sample stress, concentration, drawdown, and volatility shock calls. |
| Admin Ops | Health, services, topics, DLQs, platform config, and checklist links. |
| Observability | Links to Grafana, Jaeger, Prometheus, and correlation guidance. |

## Configuration

The React app reads Vite env vars:

```text
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_BASE_URL=ws://localhost:8080
VITE_GRAFANA_URL=http://localhost:3000
VITE_JAEGER_URL=http://localhost:16686
VITE_PROMETHEUS_URL=http://localhost:9090
VITE_DEFAULT_TENANT_ID=default-tenant
```

See `frontend/trading-dashboard-react/.env.example`.

## Token Handling

The dashboard uses token paste mode for demos. Paste a JWT into the Access panel, save it to `localStorage`, and the API client sends:

- `Authorization: Bearer <token>`
- `X-Tenant-ID`
- `X-Correlation-ID`

Clear removes the saved token.

## WebSocket Panel

The Realtime page connects to:

- `/ws`
- `/ws/orders`
- `/ws/alerts`
- `/ws/notifications`
- `/ws/audit`
- `/ws/market`

The client passes `token` as a query parameter and reconnects with a small backoff. Heartbeat and connection messages appear in the event list.

## Local Run

```bash
cd frontend/trading-dashboard-react
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

Docker Compose serves the built dashboard at `http://localhost:4300`.

## Limitations

- No full login flow; paste a JWT from the existing auth API.
- No chart library; cards/tables keep the demo light.
- No Redux or complex client cache.
- Runtime env changes require rebuilding the static Docker image.
