# Resilience Patterns

v1.2.0 adds practical failure handling to the event-driven services and API Gateway without changing core business behavior.

## Retry With Backoff

Event consumers retry processing failures before committing the source message:

- `portfolio-service`: `order.filled`
- `surveillance-service`: surveillance source topics
- `notification-service`: surveillance alert lifecycle topics

Environment configuration:

```bash
EVENT_PROCESSING_MAX_RETRIES=3
EVENT_PROCESSING_RETRY_BACKOFF_MS=500
EVENT_PROCESSING_RETRY_BACKOFF_MULTIPLIER=2
```

With the defaults, a failed event is attempted once, retried up to three times, then written to the service DLQ.

## Dead-Letter Topics

Failed events are published to:

- `portfolio.dlq`
- `surveillance.dlq`
- `notification.dlq`

Each DLQ record includes the original topic, original payload, error message, service name, failure timestamp, correlation ID, and retry count.

## Idempotency

The platform avoids duplicate side effects for common replay scenarios:

- Portfolio uses `processed_order_events` so the same order-fill event does not update positions twice.
- Surveillance skips duplicate alerts for the same source topic, rule, and entity.
- Notification skips duplicate notifications for the same `sourceEventId`, source event type, and channel.

## API Gateway Upstream Protection

API Gateway proxy calls use a configurable timeout:

```bash
PROXY_TIMEOUT_MS=10000
```

Upstream outcomes:

- Timeout returns `504` with `UPSTREAM_TIMEOUT`.
- Unavailable upstream returns `502` with `UPSTREAM_UNAVAILABLE`.
- The gateway preserves the request correlation ID in the response.

## Metrics

Check these metrics during demos or debugging:

- `portfolio_events_retried_total`
- `portfolio_events_deadlettered_total`
- `portfolio_event_processing_attempts_total`
- `portfolio_duplicate_events_skipped_total`
- `surveillance_events_retried_total`
- `surveillance_events_deadlettered_total`
- `surveillance_event_processing_attempts_total`
- `surveillance_event_processing_duration_seconds`
- `surveillance_duplicate_events_skipped_total`
- `notification_events_retried_total`
- `notification_events_deadlettered_total`
- `notification_event_processing_attempts_total`
- `notification_event_processing_duration_seconds`
- `notification_duplicate_events_skipped_total`
- `tradeops_api_gateway_proxy_upstream_errors_total`
- `tradeops_api_gateway_proxy_upstream_timeouts_total`

## Local Checks

```bash
curl http://localhost:8080/metrics | grep tradeops_api_gateway_proxy
curl http://localhost:8087/metrics | grep portfolio_events
curl http://localhost:8090/metrics | grep surveillance_events
curl http://localhost:8091/metrics | grep notification_events
```
