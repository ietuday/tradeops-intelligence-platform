# TradeOps Metrics Catalog

This catalog lists the Prometheus metrics currently exposed by the local TradeOps services. Metric names are intentionally kept close to the service implementation so dashboards can be traced back to code quickly during demos and interviews.

## API Gateway

| Metric | Type | Purpose |
| --- | --- | --- |
| `tradeops_api_gateway_http_requests_total` | Counter | Gateway request volume by method, route, and status code. |
| `tradeops_api_gateway_http_request_duration_seconds_bucket` | Histogram | Gateway latency distribution for p95/p99 queries. |
| `tradeops_api_gateway_proxy_upstream_errors_total` | Counter | Proxy errors returned by backend service calls. |
| `tradeops_api_gateway_proxy_upstream_timeouts_total` | Counter | Gateway upstream timeout events. |

## Identity

| Metric | Type | Purpose |
| --- | --- | --- |
| `tradeops_identity_service_http_requests_total` | Counter | Identity HTTP request volume by method, route, and status code. |
| `tradeops_identity_service_http_request_duration_seconds_bucket` | Histogram | Identity HTTP latency distribution. |

## Trading Domain Services

| Service | Metrics |
| --- | --- |
| Market Data | `market_ticks_received_total`, `market_ticks_valid_total`, `market_ticks_invalid_total`, `market_ticks_published_total`, `market_tick_processing_duration_seconds_bucket`, `mqtt_connection_status`, `kafka_publish_errors_total` |
| Order | `orders_created_total`, `orders_accepted_total`, `orders_filled_total`, `orders_rejected_total`, `orders_cancelled_total`, `idempotency_replays_total`, `order_processing_duration_seconds_bucket`, `kafka_publish_errors_total` |
| Portfolio | `portfolio_updates_total`, `portfolio_update_failures_total`, `portfolio_holdings_count`, `portfolio_cash_balance`, `portfolio_realized_pnl_total`, `portfolio_unrealized_pnl_total`, `portfolio_event_processing_attempts_total`, `portfolio_events_retried_total`, `portfolio_events_deadlettered_total`, `portfolio_duplicate_events_skipped_total`, `portfolio_event_processing_duration_seconds_bucket` |
| Strategy | `strategies_created_total`, `backtests_started_total`, `backtests_completed_total`, `backtests_failed_total`, `strategy_signals_generated_total`, `backtest_duration_seconds_bucket`, `kafka_publish_errors_total` |
| Risk | `risk_scores_calculated_total`, `risk_breaches_total`, `risk_anomalies_detected_total`, `risk_recommendations_created_total`, `risk_calculation_duration_seconds_bucket`, `risk_score_current`, `var_current`, `drawdown_current`, `kafka_publish_errors_total` |

## Surveillance

| Metric | Type | Purpose |
| --- | --- | --- |
| `surveillance_alerts_created_total` | Counter | Alerts created by rules. |
| `surveillance_alerts_acknowledged_total` | Counter | Alert acknowledgement transitions. |
| `surveillance_alerts_resolved_total` | Counter | Alert resolution transitions. |
| `surveillance_alerts_dismissed_total` | Counter | Alert dismissal transitions. |
| `surveillance_rule_matches_total` | Counter | Rule matches by rule name. |
| `surveillance_rule_executions_total` | Counter | Rule execution volume by rule and topic. |
| `surveillance_kafka_messages_total` | Counter | Consumed Kafka message volume. |
| `surveillance_event_processing_attempts_total` | Counter | Consumer processing attempts by topic and status. |
| `surveillance_events_retried_total` | Counter | Retried surveillance events. |
| `surveillance_events_deadlettered_total` | Counter | Events published to surveillance DLQ. |
| `surveillance_duplicate_events_skipped_total` | Counter | Idempotent duplicate skips. |
| `surveillance_event_processing_duration_seconds_bucket` | Histogram | Surveillance event processing latency. |

## Notification

| Metric | Type | Purpose |
| --- | --- | --- |
| `notification_events_processed_total` | Counter | Successfully processed notification source events. |
| `notification_events_failed_total` | Counter | Failed notification event processing. |
| `notifications_created_total` | Counter | Notifications created from event processing or APIs. |
| `notifications_marked_read_total` | Counter | Notifications marked read by users. |
| `notification_retries_total` | Counter | Retry requests. |
| `notification_preferences_updated_total` | Counter | Preference updates. |
| `notification_delivery_attempts_total` | Counter | Delivery attempts by sender/channel. |
| `notification_delivery_failures_total` | Counter | Failed delivery attempts. |
| `notification_status_updates_total` | Counter | Notification status transitions. |
| `notification_delivery_duration_seconds_bucket` | Histogram | Delivery duration by channel. |
| `notification_event_processing_attempts_total` | Counter | Consumer attempts by topic and status. |
| `notification_events_retried_total` | Counter | Retried notification source events. |
| `notification_events_deadlettered_total` | Counter | Events published to notification DLQ. |
| `notification_duplicate_events_skipped_total` | Counter | Duplicate notification source events skipped. |

## Audit

| Metric | Type | Purpose |
| --- | --- | --- |
| `audit_events_processed_total` | Counter | Source events processed by topic. |
| `audit_events_failed_total` | Counter | Failed audit event processing. |
| `audit_logs_created_total` | Counter | Audit logs created by service, event type, and severity. |
| `audit_logs_duplicate_skipped_total` | Counter | Duplicate audit source events skipped. |
| `audit_events_deadlettered_total` | Counter | Audit events sent to `audit.dlq`. |
| `audit_events_retried_total` | Counter | Audit event retry attempts. |
| `audit_event_processing_attempts_total` | Counter | Audit consumer attempts by topic and status. |
| `audit_event_processing_duration_seconds_bucket` | Histogram | Audit event processing latency. |
| `audit_export_requests_total` | Counter | Audit export requests by format. |
| `audit_kafka_publish_errors_total` | Counter | Audit event publish failures. |

## Dashboard Query Pattern

Dashboards use `or vector(0)` for optional metrics so an empty local demo does not render broken panels before events have been generated. This is a demo-friendly fallback, not a substitute for production cardinality and scrape health reviews.

