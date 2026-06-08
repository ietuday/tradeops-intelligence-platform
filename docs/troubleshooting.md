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

## Webhook Delivery Failed

Symptom: Notification status is `FAILED` or delivery attempts show non-2xx responses.

Possible cause: Webhook URL is unreachable from Docker, returns an error, or times out.

Useful command:

```bash
curl http://localhost:8080/api/notifications/preferences -H "Authorization: Bearer ${TOKEN}"
docker compose -f infrastructure/docker/docker-compose.yml logs notification-service
```

Fix: Use a reachable webhook endpoint, return a 2xx response, and then call `/api/notifications/{id}/retry` for failed notifications.

## Prometheus Target Down

Symptom: Prometheus target page shows a service as down.

Possible cause: Service is unhealthy, wrong scrape target, or service metrics endpoint is unavailable.

Useful command:

```bash
curl http://localhost:9090/targets
curl http://localhost:8091/metrics
```

Fix: Confirm the service is running and that `infrastructure/docker/prometheus/prometheus.yml` points at the correct Compose service name and port.

## Grafana Dashboard Not Loading

Symptom: Grafana starts but TradeOps dashboards are missing.

Possible cause: Provisioning path is wrong, dashboard JSON is invalid, or Grafana has not reloaded provisioning.

Useful command:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs grafana
find infrastructure/docker/grafana -type f
```

Fix: Restart Grafana and confirm dashboard JSON files are mounted under the configured provisioning path.

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
