# Observability Runbook

Use this runbook when dashboards look empty, alerts fire, or a local demo needs a quick health check.

## Quick Checks

```bash
docker compose -f infrastructure/docker/docker-compose.yml ps
curl -fsS http://localhost:9090/-/ready
curl -fsS http://localhost:9090/api/v1/targets
curl -fsS http://localhost:9090/api/v1/rules
```

Open:

```text
Prometheus: http://localhost:9090
Grafana:    http://localhost:3000
```

## Service Down

1. Check Compose status with `docker compose -f infrastructure/docker/docker-compose.yml ps`.
2. Inspect the affected service logs.
3. Check direct `/health` and `/ready`.
4. Check Prometheus target labels to confirm the scrape target matches the Compose service name and port.

## Dashboard Has No Data

1. Confirm Prometheus is ready.
2. Open Prometheus targets and verify each backend target is up.
3. Run a focused demo to generate traffic or events.
4. Query the metric directly in Prometheus.
5. Restart Grafana if dashboard provisioning has just changed.

## Alert Rule Missing

1. Confirm `prometheus.yml` includes `rule_files`.
2. Confirm Compose mounts `./prometheus/rules:/etc/prometheus/rules:ro`.
3. Open `http://localhost:9090/rules`.
4. Run `docker compose -f infrastructure/docker/docker-compose.yml config` to catch YAML mistakes.

## DLQ Or Event Failure Alert

1. Check service logs for payload validation or dependency errors.
2. Inspect the matching DLQ topic with `rpk topic consume`.
3. Compare the failed payload to examples under `docs/examples/`.
4. Replay only known-safe corrected events.

## Notification Delivery Alert

1. Check `notification-service` logs.
2. Confirm webhook preferences point to a URL reachable from Docker.
3. Check `notification_delivery_attempts_total` and `notification_delivery_failures_total`.
4. Retry failed notifications only after the endpoint is healthy.

## Audit Ingestion Alert

1. Check `audit-service` logs.
2. Inspect `audit_events_failed_total` and `audit_events_deadlettered_total`.
3. Verify the source topic and payload shape.
4. Use `./scripts/demo-audit.sh` with known-good payloads to compare behavior.

