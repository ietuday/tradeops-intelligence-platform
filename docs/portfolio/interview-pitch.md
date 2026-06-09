# Interview Pitch

## 30-Second Pitch

TradeOps is an event-driven trading microservices platform I built to demonstrate senior backend engineering. It uses Go, Python FastAPI, Node.js, PostgreSQL, Redis, Redpanda/Kafka, MQTT, Prometheus, Grafana, Docker Compose, and optional Helm templates to model order flow, portfolio updates, risk, surveillance, notifications, and audit trails.

## 60-Second Pitch

TradeOps simulates a trading operations backend. The API Gateway exposes stable routes, identity issues JWTs, the order service handles idempotent order creation, portfolio consumes fills, surveillance detects alerts from order/market/risk events, notifications are created from alert lifecycle events, and audit records compliance-style logs. I added reliability patterns like retries and DLQ guidance, observability with Prometheus/Grafana/SLO docs, security docs and gateway hardening, backup/replay scripts, Helm deployment readiness, and local performance testing.

v2.2.0 adds shared-database multitenancy: JWT `tenantId`, `X-Tenant-ID` propagation, tenant-aware database columns, tenant-aware events, audit logs, and WebSocket filtering.

## 2-Minute Pitch

The project is intentionally built like a local version of a production backend platform. Synchronous HTTP handles commands and queries, while Kafka-compatible Redpanda handles asynchronous workflows. Services own their data and expose health, readiness, and metrics. The system includes realistic concerns: idempotency for order creation, defensive event consumers, retry/DLQ documentation, audit export, notification delivery attempts, correlation IDs across HTTP/events/logs, and dashboards for latency, errors, and event processing.

The goal was not to build a toy trading UI, but to show platform engineering judgment: clear service boundaries, operational docs, validation scripts, demo flows, security threat modeling, capacity planning, and honest limitations.

## Deep Technical Walkthrough

Start with the API Gateway. It centralizes client access, forwards authorization and correlation headers, applies Helmet/CORS/body limits/rate limiting, and proxies to service-owned APIs. Identity issues JWTs. Go services handle transactional domains; Python services handle analytics-oriented strategy and risk; Node.js keeps the gateway lightweight.

Events connect the system. Orders produce lifecycle topics, portfolio updates positions, risk emits scores, surveillance applies rules, notifications respond to alert lifecycle events, and audit normalizes important events into searchable logs. The platform uses sample payloads and replay scripts for repeatable demos.

## System Design Explanation

TradeOps uses a gateway plus domain services. PostgreSQL stores durable state. Redis supports identity session state. MQTT is the simulated raw market input. Redpanda/Kafka decouples services and makes event-driven workflows visible. Prometheus/Grafana provide local operational visibility.

## Event-Driven Architecture Explanation

Kafka topics are used where services should not block each other synchronously. For example, order creation can publish events consumed by portfolio, surveillance, and audit independently. This keeps service ownership clear and supports replay/DLQ workflows.

## Reliability Explanation

The system includes idempotency for order creation, retry/backoff patterns in event consumers, DLQ documentation, conservative replay scripts, smoke tests, health/readiness endpoints, and graceful shutdown guidance.

## Observability Explanation

Each backend exposes `/health`, `/ready`, and `/metrics`. Prometheus scrapes services, Grafana dashboards show gateway latency/errors and event processing health, and correlation IDs connect HTTP requests to events/logs/audit records.

## Security Explanation

Identity issues JWTs and services enforce RBAC where implemented. The gateway applies practical controls such as security headers, CORS config, body limits, and rate limiting. Security docs include a STRIDE threat model, RBAC matrix, API security notes, and secrets guidance.

## Multitenancy Explanation

The platform uses shared PostgreSQL tables with `tenant_id` for a simple portfolio-scale tenant model. It avoids database-per-tenant operational overhead while still showing tenant propagation, query filtering, tenant-aware audit, and real-time stream isolation.

## Deployment Explanation

Docker Compose is the primary runtime. The optional Helm chart shows Kubernetes packaging readiness with Deployments, Services, ConfigMap/Secret separation, probes, and resource requests/limits, without pretending to be a complete production cluster.
