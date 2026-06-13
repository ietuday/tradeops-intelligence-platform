# Dashboard Demo Guide

## Start Backend

```bash
cp infrastructure/docker/.env.example infrastructure/docker/.env
make dev-up
make smoke
```

## Start Frontend

```bash
./scripts/demo-dashboard.sh
cd frontend/trading-dashboard-react
npm install
npm run dev
```

Open `http://localhost:5173`.

## Demo Flow

1. Get or paste a JWT with `trading_admin` or `risk_manager`.
2. Save the token in the Access panel.
3. Open Dashboard and confirm platform/admin summaries load.
4. Open Admin Ops and show health, services, topics, DLQs, and masked config.
5. Open Realtime, select `alerts` or `orders`, and connect.
6. Publish events with existing demo scripts such as `demo-websocket-streams.sh`.
7. Open Risk Analytics and run the sample stress test.
8. Run concentration, drawdown, and volatility shock samples.
9. Open Observability and jump to Grafana or Jaeger.

## Screenshots

Suggested captures:

- Dashboard home with platform status.
- Realtime page with connection.ready or heartbeat message.
- Risk Analytics page after a stress test result.
- Admin Ops service health and topic table.
- Observability links with Grafana or Jaeger open beside it.

Do not capture visible JWTs or local secrets.
