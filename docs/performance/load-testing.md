# Load Testing

TradeOps load tests are local, safe-by-default checks for demo confidence and bottleneck discovery. They are not production capacity numbers.

## Safety Notes

- Local results depend on CPU, RAM, disk speed, Docker resource limits, and background processes.
- Keep default k6 settings low: `5` virtual users for `30s`.
- Stop tests if Docker Desktop, the kernel, or laptop fans show sustained pressure.
- Do not run destructive database cleanup while load tests are active.

## Prerequisites

```bash
make dev-up
make smoke
```

Optional k6:

```bash
k6 version
```

If k6 is not installed, `./scripts/run-load-tests.sh` exits successfully and prints install guidance.

## Lightweight Smoke

```bash
./scripts/perf-smoke.sh
```

This uses only `curl`, sends a small number of health requests, and prints `service`, `endpoint`, `status`, and `time_total`.

## API Gateway Test

```bash
./scripts/run-load-tests.sh --gateway
```

This checks `/health` and `/ready` through the gateway.

## Order Creation Test

```bash
TOKEN=<jwt> ./scripts/run-load-tests.sh --orders
```

The order scenario posts a small market order to `/api/orders` with an idempotency key. If `TOKEN` is missing, the script prints a helpful message and skips protected order creation.

## Surveillance Alert Flow

```bash
TOKEN=<jwt> ./scripts/run-load-tests.sh --surveillance
```

This checks `/api/surveillance/health` and lists alerts when a token is available. For event-driven alert creation under load, use `./scripts/replay-sample-events.sh --surveillance` in a separate terminal at a low rate and watch surveillance metrics.

## Notification Listing

```bash
TOKEN=<jwt> ./scripts/run-load-tests.sh --notifications
```

This checks `/api/notifications/health` and lists notifications when a token is available.

## Audit Log Listing

```bash
TOKEN=<jwt> ./scripts/run-load-tests.sh --audit
```

This checks `/api/audit/health` and lists audit logs when a token is available. Audit export should be tested sparingly because exports can be heavier than paginated searches.

## Sample Event Replay Under Load

Use sample replay conservatively:

```bash
./scripts/replay-sample-events.sh --surveillance
./scripts/replay-sample-events.sh --notifications
./scripts/replay-sample-events.sh --audit
```

Watch event processing metrics, DLQ metrics, service logs, and PostgreSQL pressure while replay is running.

## Stop Tests Safely

- Press `Ctrl+C` in the terminal running k6.
- Stop the stack with `make dev-down` after collecting results.
- If Docker is under pressure, stop load first, then stop demos/scripts.

## Interpreting Results

- p95 latency shows the slow tail users feel during local demos.
- Error rate should stay below the scenario threshold.
- Throughput is useful for comparing local changes on the same machine, not for production sizing.
- Watch Prometheus/Grafana at the same time for gateway latency, 5xx rate, event failures, retries, DLQ events, and delivery failures.

## Known Limitations

- Docker Compose runs all dependencies on one machine.
- In-memory rate limiting and local Postgres/Redpanda do not model production topology.
- Kafka consumer lag metrics are not fully instrumented yet.
- Results are best used as relative baselines across commits on the same machine.
