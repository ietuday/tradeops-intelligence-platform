# Screenshot Guide

No screenshots are committed yet. Before publishing the GitHub repository or portfolio page, consider adding screenshots for:

- Docker Compose services running.
- Prometheus targets page with platform services healthy.
- Grafana TradeOps Platform Overview dashboard.
- Grafana TradeOps Events And Alerts dashboard.
- Redpanda Console topics list.
- API Gateway `/health` response.
- Surveillance alert API response from `/api/surveillance/alerts`.
- Notification API response from `/api/notifications`.
- End-to-end demo script output from `./scripts/demo-e2e-tradeops.sh`.

Suggested folder structure:

```text
docs/screenshots/
  docker-compose-services.png
  prometheus-targets.png
  grafana-platform-overview.png
  grafana-events-alerts.png
  redpanda-topics.png
  api-gateway-health.png
  surveillance-alerts.png
  notifications.png
  e2e-demo-output.png
```

Keep screenshots free of local secrets, tokens, email addresses, and private machine paths before committing them.
