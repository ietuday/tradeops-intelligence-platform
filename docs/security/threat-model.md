# Threat Model

This STRIDE-style model describes the local TradeOps Intelligence Platform and the practical controls currently in place. It is intended for portfolio, interview, and release-readiness discussion.

## System Overview

TradeOps exposes client traffic through the API Gateway. Backend services own identity, market data, orders, portfolio, strategy, risk, surveillance, notifications, and audit. PostgreSQL stores service data, Redis supports identity state, Redpanda/Kafka carries business events, Mosquitto receives market ticks, and Prometheus/Grafana provide observability.

## Trust Boundaries

| Boundary | Notes |
| --- | --- |
| External client to API Gateway | Public HTTP entry point, CORS, request size limits, rate limiting, security headers, tenant context, correlation IDs. |
| API Gateway to internal services | Internal Compose/Kubernetes service URLs; authorization, tenant, and correlation headers are forwarded. |
| Services to PostgreSQL | Service-owned tables share local PostgreSQL for convenience; production should isolate credentials/schemas. |
| Services to Redis | Identity refresh/session state uses Redis in local Compose. |
| Services to Redpanda/Kafka | Events cross service boundaries; consumers validate payloads defensively. |
| Services to MQTT broker | Market ticks enter through Mosquitto before normalization. |
| Prometheus/Grafana access | Local observability endpoints are useful for demos; production access needs authentication and network controls. |
| Optional Kubernetes cluster boundary | Helm chart renders application services only; infrastructure and ingress security remain operator responsibilities. |

## Assets

- JWT tokens and signing secrets.
- User accounts and role assignments.
- Orders, portfolio data, market data, risk scores, surveillance alerts, notifications, and audit logs.
- Secrets and runtime configuration.
- Kafka topics, DLQ records, backup files, and replay payloads.

## Threats And Mitigations

| STRIDE Area | Example Threats | Current Mitigations |
| --- | --- | --- |
| Spoofing | Spoofed JWT, forged service requests. | JWT validation, shared local secret, API Gateway boundary, correlation IDs. |
| Tampering | Replayed idempotency keys/events, Kafka topic misuse, malformed payloads. | Idempotency on order creation, defensive consumers, DLQ/retry, audit trail. |
| Repudiation | Missing action history, missing correlation visibility. | Audit logs, `X-Correlation-ID`, event `correlationId`, structured logs. |
| Information Disclosure | Unauthorized audit export, leaked `.env` secrets, overly broad CORS. | RBAC middleware where implemented, ignored local env files, env-driven CORS, secrets examples only. |
| Denial Of Service | Request flooding, oversized payloads, webhook abuse. | API Gateway rate limiting, request body limit, proxy timeouts, webhook timeout/retry controls. |
| Elevation Of Privilege | Role bypass, weak demo tokens, admin action misuse. | JWT/RBAC checks where implemented, RBAC matrix documentation, production gap documentation. |
| Tenant Isolation | Cross-tenant reads, forged `X-Tenant-ID`, tenantless events. | JWT `tenantId`, gateway override rules, service tenant filters, default tenant fallback, tenant-aware audit. |

## Known Gaps

- No OAuth/OIDC provider, mTLS, external secret manager, production WAF, or managed identity provider.
- No production TLS ingress by default.
- Local Docker Compose is the default runtime.
- Kafka schemas are example-based, not schema-registry enforced.
- Notification email delivery is mock/log-only.
- Tenant isolation is shared-database/application-enforced; production may require schema/database-per-tenant for stricter contractual isolation.

## Future Improvements

- Add OIDC integration and token revocation strategy.
- Add production TLS ingress, WAF/rate-limit integration, and network policies.
- Move real secrets to a cloud secret manager, Vault, or External Secrets.
- Add schema registry or contract tests for event topics.
- Add consumer lag monitoring and deeper OpenTelemetry tracing when warranted.
