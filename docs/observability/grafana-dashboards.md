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
- During performance tests, watch the Platform Overview and API Gateway dashboards for p95 latency, 5xx rate, request volume, upstream errors, and timeouts.
- During event replay or event-driven load tests, watch the Event Processing dashboard for retries, failures, duplicate skips, and DLQ events.
- Some panels may show zero until a workflow produces matching events.
- Dashboard queries use local demo thresholds and are not tuned for a real production trading workload.

## Performance Testing Queries

Use these Prometheus starters while running `./scripts/perf-smoke.sh` or `./scripts/run-load-tests.sh --gateway`:

```promql
histogram_quantile(0.95, sum by (le) (rate(tradeops_api_gateway_http_request_duration_seconds_bucket[5m])))
sum(rate(tradeops_api_gateway_http_requests_total{status_code=~"5.."}[5m]))
sum(rate(tradeops_api_gateway_proxy_upstream_errors_total[5m]))
sum(rate(surveillance_events_deadlettered_total[5m])) + sum(rate(notification_events_deadlettered_total[5m])) + sum(rate(audit_events_deadlettered_total[5m]))
```
