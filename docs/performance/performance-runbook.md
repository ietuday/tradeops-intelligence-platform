# Performance Runbook

Use this runbook during local load tests or demo troubleshooting. Pair every investigation with service logs and Prometheus/Grafana.

| Symptom | Dashboard To Check | Prometheus Query | Logs Command | Likely Cause | Mitigation |
| --- | --- | --- | --- | --- | --- |
| API latency high | API Gateway, Platform Overview | `histogram_quantile(0.95, sum by (le) (rate(tradeops_api_gateway_http_request_duration_seconds_bucket[5m])))` | `docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway` | Upstream slow, Docker CPU pressure, DB contention. | Reduce load, inspect upstream service, tune `PROXY_TIMEOUT_MS` only after root cause. |
| API Gateway 5xx spike | API Gateway | `sum(rate(tradeops_api_gateway_http_requests_total{status_code=~"5.."}[5m]))` | `docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway` | Service unavailable, timeout, invalid upstream response. | Check `docker compose ps`, service `/ready`, and upstream logs. |
| Kafka consumer lag increasing | Event Processing | Consumer lag metric when available; proxy with retry/DLQ counters for now. | `docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service notification-service audit-service` | Consumer too slow, malformed payloads, DB unavailable. | Reduce publish rate, inspect retries, add partitions/consumers in production. |
| DLQ messages increasing | Event Processing | `sum(rate(surveillance_events_deadlettered_total[5m])) + sum(rate(notification_events_deadlettered_total[5m])) + sum(rate(audit_events_deadlettered_total[5m]))` | `docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service notification-service audit-service` | Bad payload or repeated dependency failure. | Stop replay/load, inspect DLQ payload, replay only after fix. |
| PostgreSQL slow queries | Platform Overview, service dashboards | Service latency histograms plus DB logs. | `docker compose -f infrastructure/docker/docker-compose.yml logs postgres` | Missing index, export scan, write contention, Docker disk pressure. | Limit exports, add indexes in future, use managed DB/connection pooling for production. |
| Notification webhook timeout spike | Surveillance & Notifications | `sum(rate(notification_delivery_failures_total[5m]))` | `docker compose -f infrastructure/docker/docker-compose.yml logs notification-service` | Webhook endpoint unreachable or slow. | Disable failing preference, lower test rate, verify URL from Docker network. |
| Audit export slow | Audit & Compliance | `sum(rate(audit_export_requests_total[5m]))` and gateway p95 latency. | `docker compose -f infrastructure/docker/docker-compose.yml logs audit-service` | Large export or broad filters. | Use smaller limits/filters; production should offload large exports. |
| Market tick ingestion falling behind | Event Processing, Platform Overview | `rate(market_ticks_received_total[5m])`, `rate(kafka_publish_errors_total[5m])` | `docker compose -f infrastructure/docker/docker-compose.yml logs market-data-service mosquitto` | MQTT surge, Kafka publish errors, Docker CPU pressure. | Reduce simulator rate, inspect broker health, partition topics in production. |
| Docker CPU/memory pressure | Docker Desktop/host monitor, Platform Overview | `up` plus service latency/error queries. | `docker compose -f infrastructure/docker/docker-compose.yml ps` | Local machine saturated by all services and dependencies. | Stop load tests, reduce k6 VUs/duration, increase Docker resources if safe. |

## During A Test

1. Run one scenario at a time.
2. Keep Prometheus and Grafana open.
3. Watch gateway p95, 5xx rate, retries, DLQ, and service logs.
4. Stop tests with `Ctrl+C` before changing configuration.
