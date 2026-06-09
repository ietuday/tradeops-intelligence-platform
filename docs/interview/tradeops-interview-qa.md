# TradeOps Interview Q&A

## Architecture

1. **What is TradeOps?**  
TradeOps is a local event-driven trading microservices platform that demonstrates backend architecture, operational readiness, security, observability, reliability, and demo workflows.

2. **Why use microservices here?**  
The domain naturally separates into identity, orders, portfolio, risk, surveillance, notifications, and audit. Separate services make ownership, event boundaries, and operational concerns easier to explain.

3. **What is the main architectural pattern?**  
A gateway-fronted microservices architecture with synchronous HTTP for commands/queries and Kafka-compatible events for asynchronous workflows.

4. **Why Docker Compose first?**  
Compose keeps the full platform runnable locally, which is ideal for demos, interviews, and repeatable validation without cloud dependencies.

5. **What would change in production?**  
Use managed PostgreSQL/Kafka/Redis, TLS ingress, external secrets, distributed rate limiting, stronger identity provider integration, and mature deployment automation.

## Go Microservices

6. **Why use Go for several services?**  
Go is a strong fit for networked services: simple concurrency, fast startup, good standard library, static binaries, and predictable runtime behavior.

7. **Which services are Go-based?**  
Identity, market data, order, portfolio, surveillance, notification, and audit are Go services.

8. **How are Go services structured?**  
They follow domain, repository, service, HTTP, config, migration, and metrics patterns where applicable.

9. **How is database access handled?**  
Go services use PostgreSQL repositories and startup migrations, keeping SQL close to service-owned domain data.

10. **How are Go services tested?**  
Unit tests focus on service logic, validation, event parsing, delivery behavior, and repository-adjacent contracts where appropriate.

## API Gateway

11. **What does the API Gateway do?**  
It exposes stable `/api/*` routes, forwards requests to internal services, propagates auth/correlation headers, exposes metrics, and applies practical hardening.

12. **Why not put all auth in the gateway?**  
The gateway is a boundary, but services still need final authorization because internal service APIs must remain safe if called directly.

13. **How are upstream failures handled?**  
Proxy helpers return consistent `502` for unavailable upstreams and `504` for timeouts, including a correlation ID.

14. **What security controls exist in the gateway?**  
Helmet headers, disabled `x-powered-by`, configurable CORS, body size limits, rate limiting, proxy timeouts, and consistent JSON errors.

15. **How is correlation handled?**  
The gateway accepts or generates `X-Correlation-ID`, returns it to clients, and forwards it downstream.

## Authentication/RBAC

16. **How does authentication work?**  
Identity issues JWTs. Protected services validate tokens using the shared local signing secret.

17. **How is RBAC modeled?**  
Roles include `trading_admin`, `trader`, `risk_manager`, `analyst`, and `viewer`, documented in the RBAC matrix.

18. **What is a known RBAC limitation?**  
Enforcement depth varies by service; the documented RBAC matrix is the target posture for production hardening.

19. **Why document target RBAC separately?**  
It makes security expectations explicit and shows where implementation gaps remain.

20. **How are secrets handled?**  
Real `.env` files are ignored; `.env.example` contains placeholders; production should use Kubernetes Secrets or an external secret manager.

## Kafka/Redpanda

21. **Why use Redpanda/Kafka?**  
It decouples services and supports asynchronous workflows like fills, surveillance alerts, notifications, and audit ingestion.

22. **What events are central?**  
Order lifecycle events, `portfolio.updated`, risk events, `surveillance.alert.*`, `notification.*`, and `audit.log.created`.

23. **How are bad events handled?**  
Consumers validate defensively, retry where implemented, and use DLQ guidance to avoid crashing on bad payloads.

24. **Why not use a schema registry yet?**  
The project keeps local scope lightweight and documents schemas with examples; schema registry is a future roadmap item.

25. **What is the replay strategy?**  
Replay is conservative: use known-good samples, inspect DLQ payloads, fix root cause first, then replay manually.

## MQTT Ingestion

26. **Why include MQTT?**  
MQTT represents raw market tick ingestion, a realistic external data source pattern in trading systems.

27. **What does market-data-service do with MQTT ticks?**  
It ingests, validates, stores or normalizes tick data, and publishes `market.ticks` to Kafka.

## Order Service

28. **What does order-service demonstrate?**  
Order validation, idempotency, lifecycle transitions, persistence, Kafka event publishing, and domain tests.

29. **Why is idempotency important for orders?**  
Clients may retry after network failures; idempotency prevents duplicate order side effects.

30. **What order payload fields matter?**  
Symbol, side, order type, quantity, and optional limit/stop prices depending on order type.

## Portfolio Service

31. **What does portfolio-service consume?**  
It consumes filled order events and updates holdings, cash, and portfolio snapshots.

32. **What consistency trade-off exists?**  
Portfolio updates are eventually consistent because they are driven by events rather than synchronous order writes.

## Strategy Service

33. **Why is strategy-service Python?**  
Strategy and analytics domains often benefit from Python’s data ecosystem and rapid iteration.

34. **What does it publish?**  
Strategy signal and backtest completion events.

## Risk Engine

35. **What does risk-engine-service calculate?**  
Risk scores, VaR, volatility, drawdown, anomalies, breaches, and recommendations.

36. **How does risk feed surveillance?**  
Risk events such as `risk.score.updated` can trigger surveillance rules like risk-score breach detection.

## Surveillance Service

37. **What does surveillance-service do?**  
It consumes order, market, portfolio, risk, and strategy events and creates rule-based alerts.

38. **What alert lifecycle is supported?**  
Alerts move from `OPEN` to `ACKNOWLEDGED`, `RESOLVED`, or `DISMISSED`.

39. **What rules are demo-ready?**  
Large order, abnormal price movement, high cancellation, rapid order, and risk-score breach style rules.

## Notification Service

40. **What events does notification-service consume?**  
Surveillance alert lifecycle topics: created, acknowledged, resolved, and dismissed.

41. **What delivery channels exist?**  
In-app, webhook, and mock/log-only email.

42. **Why is email mock-only?**  
The release avoids real providers and secrets while still demonstrating delivery abstraction.

## Audit Service

43. **What does audit-service provide?**  
Searchable audit logs, summaries, JSON/CSV export, event ingestion, idempotency, and audit metrics.

44. **Why is audit event-driven?**  
Business services do not need to block on audit writes, and audit can normalize events independently.

## Reliability/DLQ

45. **What reliability patterns are included?**  
Idempotency, retries/backoff, DLQ docs, replay scripts, graceful shutdown docs, smoke tests, and runbooks.

46. **When should DLQ events be replayed?**  
Only after understanding and fixing the cause; blind replay can duplicate side effects or repeat failures.

47. **What is a known reliability gap?**  
No production-grade automated DLQ replay service; replay remains deliberately manual and safe.

## Observability/SLOs

48. **What metrics are exposed?**  
HTTP request counts/durations, domain counters, Kafka/event attempts, retries, DLQ counters, notification delivery, and audit metrics.

49. **What dashboards exist?**  
Platform overview, API Gateway, event processing, surveillance/notifications, audit/compliance, and events/alerts.

50. **How do you trace one workflow?**  
Use `X-Correlation-ID` on HTTP, `correlationId` in events/logs, and audit log `correlation_id`.

## Security

51. **What does the threat model cover?**  
Trust boundaries, assets, STRIDE threats, mitigations, known gaps, and future improvements.

52. **What security gaps are documented?**  
No OAuth/OIDC, mTLS, external secret manager, WAF, or production TLS by default.

## Helm/Kubernetes

53. **What does the Helm chart demonstrate?**  
Deployment, Service, ConfigMap/Secret separation, probes, resources, and optional ingress for application services.

54. **Why not include stateful infrastructure in Helm?**  
Production should usually use managed or separately operated PostgreSQL, Redis, Kafka, Prometheus, and Grafana.

## Performance Testing

55. **What performance tooling exists?**  
`perf-smoke.sh` for curl timing and optional k6 scenarios for gateway, orders, surveillance, notifications, and audit.

56. **Are k6 results production capacity numbers?**  
No. They are local baselines that depend on machine resources and Docker limits.

## Trade-Offs And Limitations

57. **What is the biggest trade-off?**  
The project balances realistic architecture with local demo simplicity, so some production integrations are documented instead of implemented.

58. **Why no frontend focus?**  
The project is backend/platform focused; frontend placeholders keep attention on service design and operations.

59. **What would you improve next?**  
OpenTelemetry tracing, OIDC, mTLS, managed services, schema registry, WebSocket dashboards, and cloud deployment.

60. **How would you explain this project in an interview?**  
It is a compact but realistic backend platform showing system design, event-driven workflows, operational readiness, and honest production trade-offs.
