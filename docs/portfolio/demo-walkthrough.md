# Demo Walkthrough

## Start Docker Compose

```bash
cp infrastructure/docker/.env.example infrastructure/docker/.env
make dev-up
make ps
```

## Smoke Test

```bash
make smoke
```

## End-To-End Demo

```bash
./scripts/demo-e2e-tradeops.sh
```

Use this to tell the full story: gateway health, auth/token requirements, sample trading events, surveillance, notifications, audit, observability, and cleanup notes.

## Focused Demos

| Demo | Command | Story |
| --- | --- | --- |
| Surveillance | `./scripts/demo-surveillance.sh` | Large order event creates alert and moves through lifecycle. |
| Notifications | `./scripts/demo-notifications.sh` | Surveillance alert event creates user notification and mark-read flow. |
| Audit | `./scripts/demo-audit.sh` | Source events become searchable audit logs and export data. |
| Reliability/DLQ | `./scripts/demo-reliability.sh` | Shows defensive handling, retry/DLQ guidance, and safe replay posture. |
| Observability | `./scripts/demo-observability.sh` | Shows dashboards, Prometheus queries, and alert docs. |
| Correlation tracing | `./scripts/demo-correlation-tracing.sh` | Follows one `correlationId` through HTTP/events/logs/audit. |
| WebSocket streams | `TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --alerts` | Shows live event streaming from Kafka topics through the API Gateway. |
| Backup/replay | `./scripts/db-backup.sh`, `./scripts/replay-sample-events.sh --all` | Demonstrates operational safety and sample event replay. |
| Performance | `./scripts/perf-smoke.sh`, `./scripts/run-load-tests.sh --gateway` | Shows local-safe timing and optional k6 scenarios. |

## Explain Helm

```bash
./scripts/validate-helm.sh
helm template tradeops infrastructure/helm/tradeops-platform
```

Explain that Compose is the full local demo runtime, while Helm shows application deployment readiness and production gaps.

## Screenshots To Capture

- Docker Compose services running.
- API Gateway `/health`.
- Redpanda topics.
- Prometheus targets.
- Grafana platform and event dashboards.
- Surveillance alerts API.
- Notifications API.
- Audit logs API.
- End-to-end demo output.
- Helm template output.
- GitHub Actions passing.
