# Prometheus Alerts

Prometheus loads alert rules from:

```text
infrastructure/docker/prometheus/rules/tradeops-alerts.yml
```

The Docker Compose Prometheus service mounts the rules directory at `/etc/prometheus/rules`, and `prometheus.yml` loads `*.yml` files from that path.

## Alert Groups

| Group | Alerts |
| --- | --- |
| `tradeops-service-availability` | `ServiceDown` |
| `tradeops-api-gateway` | `HighGateway5xxRate`, `HighGatewayLatency`, `GatewayUpstreamTimeouts` |
| `tradeops-event-processing` | `EventProcessingFailures`, `DLQMessagesDetected`, `EventRetrySpike` |
| `tradeops-surveillance-notification-audit` | `SurveillanceAlertSpike`, `NotificationDeliveryFailures`, `WebhookDeliveryFailures`, `AuditIngestionFailures`, `AuditDLQMessagesDetected` |

## Local Validation

Validate Compose wiring:

```bash
docker compose -f infrastructure/docker/docker-compose.yml config
```

With the stack running, inspect loaded rules:

```text
http://localhost:9090/rules
```

Inspect active and pending alerts:

```text
http://localhost:9090/alerts
```

## Threshold Notes

The thresholds are intentionally small and demo-oriented. They are useful for detecting local failures, DLQ events, gateway errors, and obvious latency issues. A production deployment should tune thresholds from real traffic baselines and add Alertmanager routing, ownership, and escalation policies.

