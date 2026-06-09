# Benchmark Plan

Use this plan to capture repeatable local baselines. Record outcomes in [performance-results-template.md](performance-results-template.md).

| Scenario | Goal | Endpoint/Topic | Expected Behavior | Metrics To Watch | Success Criteria |
| --- | --- | --- | --- | --- | --- |
| API Gateway health check baseline | Measure gateway overhead and local HTTP latency. | `GET /health`, `GET /ready` | Stable 200 responses. | `tradeops_api_gateway_http_requests_total`, `tradeops_api_gateway_http_request_duration_seconds_bucket`, 5xx rate. | p95 under 1000ms, error rate under 5%. |
| Auth/login baseline | Validate identity/gateway auth latency. | `POST /api/auth/login` when demo credentials are available. | Login returns token or expected auth error without gateway failure. | Gateway latency, identity request latency, 4xx/5xx split. | No 5xx spike; expected 2xx/4xx behavior. |
| Order creation flow | Validate gateway, order-service, PostgreSQL, idempotency, and Kafka publish path. | `POST /api/orders`; topics `order.created`, `order.accepted`, `order.filled`. | Small market orders are accepted or fail with expected validation/auth errors. | Gateway p95, `orders_created_total`, `order_processing_duration_seconds_bucket`, `kafka_publish_errors_total`. | Low 5xx rate, no publish error spike, no unexpected duplicate behavior. |
| Event-driven alert flow | Validate surveillance rule path from order event to alert. | Publish large `order.created` sample to `order.created`. | Surveillance creates alert and emits `surveillance.alert.created`. | `surveillance_rule_matches_total`, `surveillance_alerts_created_total`, retry/DLQ metrics. | Alert created for known-good payload; failures/DLQ remain zero. |
| Notification flow | Validate alert event to notification creation. | Publish `surveillance.alert.created`; list `/api/notifications`. | Notification is created and delivery attempts are recorded. | `notifications_created_total`, `notification_delivery_attempts_total`, delivery failures, retries/DLQ. | Notification appears; expected channels succeed or fail explainably. |
| Audit search/export | Validate audit read path and export behavior. | `GET /api/audit/logs`, `GET /api/audit/export?format=json|csv`. | Filtered logs return; export works for small limits. | `audit_logs_created_total`, `audit_export_requests_total`, gateway latency, audit failures. | Search remains responsive; export is limited and role-protected. |

## Baseline Procedure

1. Start from a clean local stack: `make dev-up`.
2. Run `make smoke`.
3. Run `./scripts/perf-smoke.sh`.
4. Run one k6 scenario at a time.
5. Capture Docker resource settings and git commit.
6. Save results in a copy of the results template.
