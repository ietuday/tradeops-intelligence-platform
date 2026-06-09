# Screenshots Guide

Do not commit secrets, tokens, or private URLs in screenshots. Redact JWTs and local `.env` values.

## Suggested Screenshots

| Screenshot | What It Shows |
| --- | --- |
| Docker Compose services running | Full local platform is up. |
| API Gateway health response | Single entry point works. |
| Redpanda console topics | Event-driven backbone and topic visibility. |
| Prometheus targets | Services are scraped and observable. |
| Grafana platform dashboard | Service health, request rate, latency, and errors. |
| Grafana event processing dashboard | Retry, failure, duplicate, and DLQ visibility. |
| Surveillance alerts API | Rule-based alert output. |
| Notifications API | Alert-to-notification workflow. |
| Audit logs API | Compliance-style event trail. |
| `demo-e2e` script output | Guided platform story. |
| Helm template output | Kubernetes deployment-readiness discussion. |
| GitHub Actions passing | CI/CD quality gate evidence. |

## Tips

- Keep terminal font readable.
- Capture the command and result together.
- Prefer local demo data over personal accounts.
- Do not add actual screenshots unless they have been reviewed for secrets.
