# Troubleshooting Guide

This guide focuses on the local Docker Compose platform.

## Docker Compose Fails

Symptom: `docker compose up` exits, services stay in `Created`, or required variables are missing.

Possible cause: Missing `infrastructure/docker/.env`, Docker daemon unavailable, or invalid Compose config.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml config
docker compose -f infrastructure/docker/docker-compose.yml ps
```

Fix: Copy `infrastructure/docker/.env.example` to `infrastructure/docker/.env`, set required secrets, then rerun `make dev-up`.

## Admin Health Summary Is Degraded

Symptom: `GET /api/admin/health-summary` returns `DEGRADED` or `UNHEALTHY`.

Possible cause: A downstream `/health` endpoint is failing, timing out, or unreachable from the API Gateway network.

Useful command:

```bash
TOKEN=<jwt> ./scripts/demo-admin-ops.sh
docker compose -f infrastructure/docker/docker-compose.yml ps
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway order-service surveillance-service notification-service audit-service
```

Fix: Check the failing service listed in the admin response, verify its Compose service is healthy, and increase `ADMIN_HEALTH_TIMEOUT_MS` only if the service is healthy but slow in the local environment.

## PostgreSQL Not Ready

Symptom: Services fail readiness checks or logs mention database connection failures.

Possible cause: Postgres is still starting, the password/database URL is wrong, or migrations failed.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs postgres
docker compose -f infrastructure/docker/docker-compose.yml ps postgres
```

Fix: Confirm `POSTGRES_PASSWORD` and service database URLs match. Restart the affected service after Postgres is healthy.

## Redpanda Not Ready

Symptom: Kafka health/readiness checks fail or services cannot publish/consume events.

Possible cause: Redpanda is still starting, Docker networking is unavailable, or the broker address is wrong.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs redpanda
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk cluster info
```

Fix: Ensure services use `redpanda:29092` inside Compose. Restart event-driven services after Redpanda is reachable.

## Kafka Topic Not Receiving Event

Symptom: Publishing succeeds but consumers do not react.

Possible cause: Wrong topic, malformed payload, consumer not running, or event was published before the consumer started.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic list
docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service notification-service
```

Fix: Use known-good sample payloads under `docs/examples/`, publish to the exact expected topic, and check service logs for payload validation errors.

## Event Stuck In DLQ

Symptom: A failed event appears in `portfolio.dlq`, `surveillance.dlq`, or `notification.dlq`.

Possible cause: The event failed all retry attempts because the payload was malformed, the database was unavailable, or a downstream call failed repeatedly.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume portfolio.dlq -n 1
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume surveillance.dlq -n 1
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume notification.dlq -n 1
```

Fix: Inspect `errorMessage` and `correlationId`, fix the source issue, then manually replay `originalPayload` to `originalTopic` only when it is safe. See `docs/reliability/dead-letter-topics.md` and `docs/data-lifecycle/dlq-replay.md`.

## Backup Or Restore Failure

Symptom: `scripts/db-backup.sh` or `scripts/db-restore.sh` fails.

Possible cause: Docker is not running, the Compose postgres service is unhealthy, the backup file is missing, or restore was attempted without `--confirm`.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml ps postgres
./scripts/db-backup.sh
./scripts/db-restore.sh backups/file.sql --confirm
```

Fix: Confirm Postgres is healthy, verify the backup file exists and is non-empty, and read `docs/data-lifecycle/backup-restore.md` before restoring.

## Archive Script Finds No Rows

Symptom: `scripts/archive-old-data.sh` exports CSV files with only headers or prints zero candidate rows.

Possible cause: Local demo data is newer than the retention windows or the table does not exist in the current environment.

Useful command:

```bash
./scripts/archive-old-data.sh
docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres \
  psql -U tradeops -d tradeops -c "SELECT relname, n_live_tup FROM pg_stat_user_tables ORDER BY n_live_tup DESC;"
```

Fix: This is usually expected in a fresh local environment. Adjust retention env vars only for local testing, and use `--delete-confirm` only after validating archive output.

## Sample Replay Does Not Create Records

Symptom: `scripts/replay-sample-events.sh --all` publishes events, but consumers do not create alerts, notifications, or audit logs.

Possible cause: Redpanda is unavailable, the target consumer is unhealthy, the event was skipped as a duplicate, or the payload does not match the expected user/context.

Useful command:

```bash
./scripts/replay-sample-events.sh --all
docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service notification-service audit-service
```

Fix: Confirm Redpanda and consumers are healthy, inspect duplicate-skip metrics, and replay one known-good payload at a time.

## Surveillance Rule Does Not Trigger

Symptom: A known-good event does not create the expected surveillance alert.

Possible cause: The rule is disabled, the tenant-specific threshold is higher than expected, the event belongs to another tenant, or the event was skipped as a duplicate.

Useful command:

```bash
TOKEN=<jwt> ./scripts/demo-rule-config.sh
curl http://localhost:8090/metrics | grep surveillance_rule_disabled_skips_total
```

Fix: Check `/api/surveillance/rules`, confirm `enabled=true`, verify the threshold fields, and make sure the event payload carries the expected `tenantId` and `correlationId`.

## Rule Simulation Returns Unexpected Counts

Symptom: `POST /api/surveillance/rules/{ruleName}/simulate` returns fewer or more would-trigger alerts than expected.

Possible cause: The request used the wrong tenant, the proposed threshold field does not match the rule type, the rule is disabled in the proposed config, or the simulation is using deterministic demo/historical-style events rather than live Kafka history.

Useful command:

```bash
TOKEN=<jwt> ./scripts/demo-rule-simulation.sh
curl http://localhost:8090/metrics | grep surveillance_rule_simulation
```

Fix: Confirm `tenantId` matches the JWT tenant, use canonical rule names such as `LargeOrderRule`, and compare the current/strict/relaxed demo outputs. Simulation is intentionally dry-run only: it will not create live alerts or notification records.

## Rule Simulation Fails RBAC Or Tenant Check

Symptom: Simulation returns `403 forbidden` or `401 unauthorized`.

Possible cause: Missing JWT, a token without read-oriented surveillance roles, or a request `tenantId` that does not match the JWT tenant.

Useful command:

```bash
curl -i -X POST "http://localhost:8080/api/surveillance/rules/LargeOrderRule/simulate" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"tenantId":"default-tenant","dryRun":true}'
```

Fix: Use a token with `trading_admin`, `risk_manager`, `analyst`, or `viewer`. Non-admin callers must simulate only their JWT tenant.

## Duplicate Event Detected

Symptom: A replayed event does not create a new position update, alert, or notification.

Possible cause: Idempotency checks skipped an event that was already processed.

Useful command:

```bash
curl http://localhost:8087/metrics | grep portfolio_duplicate_events_skipped_total
curl http://localhost:8090/metrics | grep surveillance_duplicate_events_skipped_total
curl http://localhost:8091/metrics | grep notification_duplicate_events_skipped_total
```

Fix: Confirm the source event ID, rule/entity, or notification channel is truly new before expecting a new side effect.

## Consumer Retry Storm

Symptom: Logs repeatedly show processing failures before events are sent to DLQ.

Possible cause: A malformed payload or unavailable dependency is causing every retry to fail.

Useful command:

```bash
curl http://localhost:8087/metrics | grep portfolio_events_retried_total
curl http://localhost:8090/metrics | grep surveillance_events_retried_total
curl http://localhost:8091/metrics | grep notification_events_retried_total
```

Fix: Reduce input noise, inspect the failed payload, and check `EVENT_PROCESSING_MAX_RETRIES`, `EVENT_PROCESSING_RETRY_BACKOFF_MS`, and `EVENT_PROCESSING_RETRY_BACKOFF_MULTIPLIER`.

## API Gateway Cannot Reach Service

Symptom: Gateway route returns `502`.

Possible cause: Upstream service is down, unhealthy, or the gateway environment URL is wrong.

Useful command:

```bash
curl http://localhost:8080/health
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway
docker compose -f infrastructure/docker/docker-compose.yml ps
```

Fix: Confirm the upstream service is healthy and the gateway has the correct `*_SERVICE_URL` value.

## Trace A Request By Correlation ID

Symptom: A request or event flow failed and you need to follow it across services.

Possible cause: The failure moved across HTTP, Kafka events, notifications, or audit ingestion.

Useful command:

```bash
CORRELATION_ID=demo-correlation-123 ./scripts/demo-correlation-tracing.sh
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway order-service surveillance-service notification-service audit-service | grep demo-correlation-123
```

Fix: Use `X-Correlation-ID` on HTTP requests, confirm downstream events include `correlationId`, inspect DLQ records for the same ID, and query audit logs with `correlationId`.

## WebSocket Stream Has No Messages

Symptom: `/ws/*` connects but only heartbeat messages appear.

Possible cause: No new Kafka events were published, the gateway Kafka consumer is not connected, or the client subscribed to the wrong stream.

Useful command:

```bash
TOKEN=<jwt> ./scripts/demo-websocket-streams.sh --alerts --publish-sample
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway
curl http://localhost:8080/metrics | grep websocket
```

Fix: Confirm `WS_ENABLED=true`, Redpanda is healthy, the token role can access the stream, and the target topic has new events.

## API Gateway Upstream Timeout

Symptom: Gateway route returns `504` with `UPSTREAM_TIMEOUT`.

Possible cause: The upstream service accepted the connection but did not respond before `PROXY_TIMEOUT_MS`.

Useful command:

```bash
curl -i http://localhost:8080/api/orders/health -H "x-correlation-id: demo-timeout"
curl http://localhost:8080/metrics | grep tradeops_api_gateway_proxy_upstream_timeouts_total
```

Fix: Check the upstream service logs and tune `PROXY_TIMEOUT_MS` only after confirming the service is healthy. The default is `10000` milliseconds.

## API Gateway Rate Limited

Symptom: A gateway route returns `429` with `RATE_LIMIT_EXCEEDED`.

Possible cause: The local in-memory gateway limiter exceeded `RATE_LIMIT_MAX_REQUESTS` within `RATE_LIMIT_WINDOW_MS`.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway
curl -i http://localhost:8080/health -H "x-correlation-id: demo-rate-limit"
```

Fix: For demos, slow the request loop or raise `RATE_LIMIT_MAX_REQUESTS` in `infrastructure/docker/.env`. For production, use a distributed gateway/WAF limit instead of the local in-memory limiter.

## API Gateway Request Body Too Large

Symptom: A gateway route returns `413` with `REQUEST_BODY_TOO_LARGE`.

Possible cause: The JSON payload exceeds `REQUEST_BODY_LIMIT`.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway
```

Fix: Reduce the payload size or adjust `REQUEST_BODY_LIMIT` for the target environment. Keep high limits behind authentication and abuse controls.

## CORS Origin Blocked

Symptom: Browser requests fail CORS checks while curl works.

Possible cause: `CORS_ORIGIN` does not include the browser application's origin.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec api-gateway env | grep CORS_ORIGIN
```

Fix: Set `CORS_ORIGIN` to a comma-separated allowlist such as `http://localhost:4200,http://localhost:4300` for local demos and the exact production UI origins for deployment.

## Security Check Warns About Secrets

Symptom: `./scripts/security-check.sh` prints `WARN` lines for secret-like strings.

Possible cause: The repository contains placeholder docs, shell variable references, or accidental credentials.

Useful command:

```bash
./scripts/security-check.sh
git ls-files | grep -E '(^|/)(\.env|.*\.pem|.*\.key|id_rsa|private_key)$'
```

Fix: Placeholder docs are expected. Remove real secrets, local `.env` files, private keys, and generated credentials before publishing.

## JWT Missing Or Invalid

Symptom: Protected API returns `401`.

Possible cause: Missing `Authorization` header, expired token, or mismatched JWT secret.

Useful command:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  --data '{"email":"demo.trader@example.com","password":"Password@123"}'
```

Fix: Login again and pass `Authorization: Bearer <token>`. For local scripted demos, set `NOTIFICATION_DEMO_TOKEN`, `SURVEILLANCE_DEMO_TOKEN`, or a shared `IDENTITY_JWT_SECRET`.

## RBAC Denied

Symptom: Protected API returns `403`.

Possible cause: Token is valid but the role is not allowed for the endpoint.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs identity-service api-gateway
```

Fix: Use an account or demo JWT with the expected role. Surveillance demos commonly require a risk/surveillance role; trading workflows commonly use `trader`.

## Surveillance Alert Not Created

Symptom: Large order event is published, but `/api/surveillance/alerts` remains empty.

Possible cause: Surveillance service is not consuming, payload is below threshold, or wrong topic was used.

Useful command:

```bash
./scripts/demo-surveillance.sh
docker compose -f infrastructure/docker/docker-compose.yml logs surveillance-service
```

Fix: Publish `docs/examples/surveillance/order-created-large-order.json` to `order.created` and confirm `SURVEILLANCE_LARGE_ORDER_THRESHOLD` is not higher than the sample order value.

## Notification Not Created

Symptom: Surveillance alert exists, but `/api/notifications` is empty.

Possible cause: Notification service is not consuming surveillance alert topics, payload user ID does not match the token user, or the notification was not sent yet.

Useful command:

```bash
./scripts/demo-notifications.sh
docker compose -f infrastructure/docker/docker-compose.yml logs notification-service
```

Fix: Publish `docs/examples/notifications/surveillance-alert-created-high.json` to `surveillance.alert.created` and list notifications with a token for the target user.

## Audit Log Not Created

Symptom: A source event was published, but `/api/audit/logs` does not show a matching audit record.

Possible cause: `audit-service` is not consuming the topic, the payload is malformed, Redpanda is unavailable, or the event was skipped as a duplicate.

Useful command:

```bash
./scripts/demo-audit.sh
docker compose -f infrastructure/docker/docker-compose.yml logs audit-service
curl http://localhost:8092/metrics | grep audit_events
```

Fix: Publish a known-good sample from `docs/examples/audit/` to the exact topic, then check `audit.dlq` and duplicate-skip metrics.

## Audit Export Or RBAC Failure

Symptom: `/api/audit/export` returns `403` or does not return the expected format.

Possible cause: The JWT role is not allowed for export, or `format` is not `json` or `csv`.

Useful command:

```bash
curl "http://localhost:8080/api/audit/export?format=csv&limit=10" \
  -H "Authorization: Bearer ${TOKEN}"
```

Fix: Use a `trading_admin` or `risk_manager` token for export. Use `trading_admin`, `risk_manager`, or `analyst` for read-only audit APIs.

## Webhook Delivery Failed

Symptom: Notification status is `FAILED` or delivery attempts show non-2xx responses.

Possible cause: Webhook URL is unreachable from Docker, returns an error, or times out.

Useful command:

```bash
curl http://localhost:8080/api/notifications/preferences -H "Authorization: Bearer ${TOKEN}"
docker compose -f infrastructure/docker/docker-compose.yml logs notification-service
```

Fix: Use a reachable webhook endpoint, return a 2xx response, and then call `/api/notifications/{id}/retry` for failed notifications.

## Webhook Timeout

Symptom: Webhook delivery attempts fail even though notification-service remains healthy.

Possible cause: The configured webhook endpoint is slow or unreachable from the Docker network.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs notification-service
curl http://localhost:8091/metrics | grep notification_delivery_failures_total
```

Fix: Test the webhook from the same network context, use a fast 2xx response for demos, and retry the notification after the endpoint is fixed.

## Prometheus Target Down

Symptom: Prometheus target page shows a service as down.

Possible cause: Service is unhealthy, wrong scrape target, or service metrics endpoint is unavailable.

Useful command:

```bash
curl http://localhost:9090/targets
curl http://localhost:8091/metrics
```

Fix: Confirm the service is running and that `infrastructure/docker/prometheus/prometheus.yml` points at the correct Compose service name and port.

## Prometheus Alert Rules Missing

Symptom: Prometheus is healthy but `http://localhost:9090/rules` does not show TradeOps alert groups.

Possible cause: Rule files are not mounted, `rule_files` is missing, or the alert YAML is invalid.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml config
docker compose -f infrastructure/docker/docker-compose.yml logs prometheus
```

Fix: Confirm `infrastructure/docker/prometheus/prometheus.yml` loads `/etc/prometheus/rules/*.yml` and Compose mounts `./prometheus/rules:/etc/prometheus/rules:ro`.

## Grafana Dashboard Not Loading

Symptom: Grafana starts but TradeOps dashboards are missing.

Possible cause: Provisioning path is wrong, dashboard JSON is invalid, or Grafana has not reloaded provisioning.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs grafana
find infrastructure/docker/grafana -type f
```

Fix: Restart Grafana and confirm dashboard JSON files are mounted under the configured provisioning path.

## Grafana Dashboard Shows No Data

Symptom: A TradeOps dashboard loads but panels are empty or show zero.

Possible cause: The stack has not generated matching traffic/events yet, Prometheus targets are down, or the selected time range does not include demo activity.

Useful command:

```bash
./scripts/demo-observability.sh
curl http://localhost:9090/api/v1/targets
```

Fix: Run `make smoke` for health traffic, then run focused demos such as surveillance, notifications, and audit to generate event metrics.

## Go Tests Fail Due Module Download

Symptom: `go test ./...` fails while downloading modules.

Possible cause: Network access is unavailable or the module cache is empty.

Useful command:

```bash
go env GOPATH GOPROXY
go test ./...
```

Fix: Restore network access or pre-populate the Go module cache, then rerun tests.

## Python Tests Fail Due Missing Venv Or Dependencies

Symptom: Python service tests fail with missing package imports.

Possible cause: Virtual environment is not active or dependencies are not installed.

Useful command:

```bash
python -m venv .venv
./.venv/bin/python -m pip install -r services/risk-engine-service/requirements.txt
```

Fix: Use the repo’s documented Python environment setup and rerun the relevant service tests.

## Port Already In Use

Symptom: Compose cannot bind ports such as `8080`, `8090`, `8091`, `9092`, or `3000`.

Possible cause: Another local process or previous Compose stack is using the port.

Useful command:

```bash
lsof -i :8080
docker compose -f infrastructure/docker/docker-compose.yml ps
```

Fix: Stop the conflicting process, change the host port mapping for local testing, or run `make dev-down` before restarting.

## Helm Validation Skipped

Symptom: `scripts/validate-helm.sh` says Helm is not installed and exits successfully.

Possible cause: Helm is not installed on the local machine.

Useful command:

```bash
./scripts/validate-helm.sh
helm version
```

Fix: Install Helm, then rerun `helm lint infrastructure/helm/tradeops-platform` and `helm template tradeops infrastructure/helm/tradeops-platform`.

## Kubernetes Pod Not Ready

Symptom: A pod from the optional Helm chart stays in `CrashLoopBackOff` or never passes readiness.

Possible cause: Placeholder secrets were not replaced, images are not available to the cluster, dependencies such as Postgres/Redis/Redpanda are missing, or `/ready` cannot reach required dependencies.

Useful command:

```bash
kubectl get pods -n tradeops
kubectl describe pod -n tradeops <pod-name>
kubectl logs -n tradeops <pod-name>
```

Fix: Confirm local images are loaded into kind/minikube, replace `JWT_SECRET=replace-me`, configure dependency endpoints in `values.yaml`, and remember that Docker Compose remains the recommended local demo runtime.

## Tenant Data Missing

Symptom: APIs return no orders, alerts, notifications, or audit logs even though data exists.

Possible cause: JWT `tenantId`, `X-Tenant-ID`, and event `tenantId` do not match, or older data was not backfilled to `default-tenant`.

Useful command:

```bash
TENANT_ID=default-tenant ./scripts/replay-sample-events.sh --all
```

Fix: Decode the JWT payload, confirm `tenantId`, check API Gateway forwarded `X-Tenant-ID`, and verify tenant migrations have run.

## No Traces In Jaeger

Symptom: Jaeger at `http://localhost:16686` has no traces for a demo request.

Possible cause: Jaeger is not running, `OTEL_ENABLED` is false, services have not restarted with OTLP config, or no gateway traffic has been generated.

Useful command:

```bash
./scripts/demo-otel-tracing.sh
docker compose --env-file infrastructure/docker/.env.example -f infrastructure/docker/docker-compose.yml logs api-gateway | grep otel
```

Fix: Start Compose with `infrastructure/docker/.env.example`, generate traffic through the API Gateway, search Jaeger for service `api-gateway`, then use span attribute `correlation.id` to grep logs or query audit records.
