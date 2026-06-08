# Grafana Dashboards

Grafana is provisioned from `infrastructure/docker/grafana/provisioning` and loads dashboard JSON files from `infrastructure/docker/grafana/dashboards`.

## Included Dashboards

| Dashboard | File | Focus |
| --- | --- | --- |
| TradeOps Platform Overview | `tradeops-platform-overview.json` | Service health, request rate, error rate, p95 latency, event processing, alerts, notifications, audit activity. |
| TradeOps API Gateway | `tradeops-api-gateway.json` | Gateway throughput, 4xx/5xx responses, p95 latency, upstream errors/timeouts, route volume. |
| TradeOps Event Processing | `tradeops-event-processing.json` | Kafka consumer success/failure, retries, DLQ events, duplicate skips, event processing latency. |
| TradeOps Surveillance & Notifications | `tradeops-surveillance-notifications.json` | Alert lifecycle, rule matches, notifications, delivery attempts, delivery failures, read/retry signals. |
| TradeOps Audit & Compliance | `tradeops-audit-compliance.json` | Audit ingestion, audit logs by service/type/severity, DLQ events, export requests, publish errors. |
| TradeOps Events & Alerts | `tradeops-events-and-alerts.json` | Earlier compact event and alert starter dashboard. |

## Local Access

Start the stack:

```bash
make dev-up
```

Open Grafana:

```text
http://localhost:3000
```

Use the local credentials from `infrastructure/docker/.env` or Compose defaults.

## Demo Notes

- Run `./scripts/demo-observability.sh` to print dashboard links and safe Prometheus queries.
- Run focused demos first if the dashboards are empty: `./scripts/demo-surveillance.sh`, `./scripts/demo-notifications.sh`, and `./scripts/demo-audit.sh`.
- Some panels may show zero until a workflow produces matching events.
- Dashboard queries use local demo thresholds and are not tuned for a real production trading workload.

