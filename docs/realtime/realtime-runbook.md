# Real-Time Streaming Runbook

## WebSocket Connection Rejected

Symptom: Client receives `401`, `403`, or connection closes during upgrade.

Check:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway
curl http://localhost:8080/metrics | grep websocket_auth_failures
```

Likely cause: Missing token, invalid JWT, unauthorized role, or origin not in `WS_ALLOWED_ORIGINS`.

Mitigation: Pass `TOKEN=<jwt>`, verify role access, and confirm origin configuration.

## Token Invalid

Symptom: Gateway logs `WebSocket authentication failed`.

Check:

```bash
echo "$TOKEN" | cut -d. -f2
```

Likely cause: Expired token, wrong local JWT secret, unsupported algorithm, or missing `roles` claim.

Mitigation: Login again through identity-service or use a token signed with the local `IDENTITY_JWT_SECRET`.

## No Messages Received

Symptom: Connection opens and heartbeat arrives, but no event messages appear.

Check:

```bash
docker compose -f infrastructure/docker/docker-compose.yml logs api-gateway
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic list
curl http://localhost:8080/metrics | grep websocket_kafka_events_consumed
```

Likely cause: Topic has no new events, Kafka consumer is not connected, client subscribed to the wrong stream, or event `tenantId` does not match the WebSocket token `tenantId`.

Mitigation: Publish a sample event with `TENANT_ID=default-tenant ./scripts/demo-websocket-streams.sh --alerts --publish-sample`.

## Tenant Filter Dropped Event

Symptom: Gateway consumes Kafka events but a specific client receives none.

Check the JWT `tenantId` and the Kafka payload `tenantId`. Non-admin clients only receive matching tenant events. `trading_admin` can receive cross-tenant events for support workflows.

## Kafka Topic Has No Events

Symptom: Redpanda is healthy but WebSocket metrics do not increase.

Check:

```bash
docker compose -f infrastructure/docker/docker-compose.yml exec redpanda rpk topic consume surveillance.alert.created -n 1
```

Mitigation: Run a focused demo or replay sample payloads.

## Too Many Connections

Symptom: Upgrade returns `429`.

Check:

```bash
curl http://localhost:8080/metrics | grep websocket_connections_active
```

Mitigation: Close stale clients or raise `WS_MAX_CONNECTIONS` for local demos.

## Client Disconnects

Symptom: Client disconnects after idle time.

Likely cause: Client/network timeout, proxy timeout, or missed heartbeat.

Mitigation: Confirm `WS_HEARTBEAT_INTERVAL_MS` and watch gateway logs for close events.

## Useful Metrics

```promql
tradeops_api_gateway_websocket_connections_active
rate(tradeops_api_gateway_websocket_messages_sent_total[5m])
rate(tradeops_api_gateway_websocket_auth_failures_total[5m])
rate(tradeops_api_gateway_websocket_kafka_events_consumed_total[5m])
```
