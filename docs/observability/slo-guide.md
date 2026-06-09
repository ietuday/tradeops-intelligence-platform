# TradeOps SLO Guide

This guide defines practical local SLO-style targets for explaining TradeOps observability. They are not contractual production SLOs.

## Suggested Service Level Indicators

| Area | SLI | PromQL Starter |
| --- | --- | --- |
| Availability | Service scrape target is up | `avg_over_time(up[5m])` |
| Gateway success rate | Non-5xx gateway responses over total gateway responses | `1 - (sum(rate(tradeops_api_gateway_http_requests_total{status_code=~"5.."}[5m])) / sum(rate(tradeops_api_gateway_http_requests_total[5m])))` |
| Gateway latency | p95 request latency | `histogram_quantile(0.95, sum by (le) (rate(tradeops_api_gateway_http_request_duration_seconds_bucket[5m])))` |
| Event processing success | Successful consumer attempts over total attempts | `sum(increase(surveillance_event_processing_attempts_total{status="success"}[5m])) / sum(increase(surveillance_event_processing_attempts_total[5m]))` |
| DLQ rate | Dead-lettered events over time | `sum(rate(surveillance_events_deadlettered_total[5m])) + sum(rate(notification_events_deadlettered_total[5m])) + sum(rate(audit_events_deadlettered_total[5m]))` |
| Notification delivery health | Delivery failures over attempts | `sum(rate(notification_delivery_failures_total[5m])) / sum(rate(notification_delivery_attempts_total[5m]))` |
| Audit ingestion health | Audit failures and DLQ events | `sum(rate(audit_events_failed_total[5m])) + sum(rate(audit_events_deadlettered_total[5m]))` |

## Demo Targets

| Area | Demo Target |
| --- | --- |
| Service availability | Core service targets are `up == 1`. |
| Gateway errors | 5xx rate remains near zero during normal demos. |
| Gateway latency | p95 remains below 500ms during local scripted demos. |
| Event processing | Failed attempts and DLQ events remain zero for known-good payloads. |
| Notifications | In-app and mock email delivery should succeed; webhook failures should be explainable by endpoint availability. |
| Audit | Known-good sample events create audit logs without DLQ entries. |

## Load Test Interpretation

- Use `./scripts/perf-smoke.sh` for a lightweight timing check before demos.
- Use k6 scenarios as local baseline tools, not production SLO proof.
- During k6 runs, p95 latency and 5xx rate are the first two gateway indicators to watch.
- For event-driven tests, retries, failures, and DLQ counters are more useful than raw request throughput.
- Keep local p95 under 1000ms for low default k6 scenarios unless the host machine is resource constrained.

## Production Follow-Ups

- Add per-route and per-service ownership for SLOs.
- Add Alertmanager routes and escalation policies.
- Add consumer lag metrics when Kafka lag instrumentation is introduced.
- Add distributed tracing for cross-service latency attribution.
- Replace local thresholds with baselines from real traffic.
