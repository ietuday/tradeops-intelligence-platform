# Capacity Planning

TradeOps capacity planning starts with the workflow: HTTP commands and queries, Kafka event throughput, PostgreSQL writes/reads, and webhook/audit side effects.

## Capacity Dimensions

| Dimension | What To Measure |
| --- | --- |
| API request throughput | Requests per second, p50/p95/p99 latency, 4xx/5xx split, route volume. |
| Kafka event throughput | Messages produced/consumed per topic, retries, DLQ events, duplicate skips. |
| PostgreSQL usage | Connection count, slow queries, index usage, write volume, migration time. |
| Redis usage | Identity refresh/session access latency and memory pressure. |
| Redpanda throughput | Topic partitioning, broker CPU/memory, producer errors, consumer processing time. |
| CPU/memory | Container pressure, garbage collection, Python worker pressure, Docker resource limits. |

## Local Docker Compose Limits

Compose co-locates services and stateful dependencies on one machine. This is excellent for demos and relative baselines, but it does not represent production capacity. Docker CPU/RAM limits, volume performance, and host contention dominate results.

## Helm Resource Requests And Limits

The optional Helm chart includes modest resource defaults. Treat them as starting points:

- Increase requests only after measuring steady-state needs.
- Increase limits only when throttling or OOM risk is understood.
- Load test before increasing replicas so bottlenecks are clear.

## Scaling Candidates

| Component | Scaling Notes |
| --- | --- |
| `api-gateway` | Horizontally scalable for stateless proxy traffic; production rate limiting should be distributed. |
| `market-data-service` | Scale by symbol/topic partition ownership and MQTT ingestion strategy. |
| `order-service` | Scale carefully around idempotency, DB writes, and Kafka publish guarantees. |
| `portfolio-service` | Scale by user/account or Kafka partition strategy to avoid conflicting updates. |
| `surveillance-service` | Scale by topic partition and rule execution cost. |
| `notification-service` | Scale by event partition and webhook delivery concurrency. |
| `audit-service` | Scale ingestion separately from search/export if audit volume grows. |

## Stateful Bottlenecks

| Dependency | Bottleneck Risk | Production Direction |
| --- | --- | --- |
| PostgreSQL | Connection exhaustion, slow writes, export scans, missing indexes. | Managed PostgreSQL, connection pooling, query tuning, read replicas where appropriate. |
| Redpanda/Kafka | Topic partition limits, broker CPU, consumer lag. | Managed Kafka/Redpanda, partition planning, lag monitoring, DLQ policy. |
| Redis | Memory pressure or network latency. | Managed Redis, eviction policy, auth/TLS, monitoring. |

## Production Recommendations

- Use managed PostgreSQL and managed Kafka/Redpanda where possible.
- Horizontally scale stateless services behind a real ingress/gateway.
- Partition Kafka topics by entity that preserves ordering needs.
- Add indexes and query optimization for hot filters.
- Use connection pooling and per-service DB credentials.
- Scale from SLOs: p95 latency, error rate, event lag, retry/DLQ rate, and webhook timeout rate.
