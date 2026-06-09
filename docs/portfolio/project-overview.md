# TradeOps Project Overview

TradeOps Intelligence Platform is an enterprise-style event-driven trading microservices platform built as a senior backend portfolio project. It models a realistic trading operations environment while remaining runnable locally with Docker Compose.

## Why It Was Built

The project demonstrates backend engineering depth beyond CRUD: service boundaries, asynchronous workflows, reliability patterns, security hardening, observability, auditability, deployment readiness, and interview-ready documentation.

## Business Domain

TradeOps simulates a trading intelligence platform with market data ingestion, order lifecycle management, portfolio updates, risk scoring, trade surveillance, notifications, and compliance-style audit logs.

## Architecture Style

- API Gateway for external HTTP entry.
- Service-owned domains for identity, market data, orders, portfolio, strategies, risk, surveillance, notifications, and audit.
- PostgreSQL for durable service data.
- Redis for identity refresh/session state.
- MQTT for market tick ingestion.
- Redpanda/Kafka for domain events.
- Prometheus/Grafana for metrics and dashboards.

## Core Services

| Service | Role |
| --- | --- |
| API Gateway | Routes external API traffic and applies gateway hardening. |
| Identity | User registration, login, JWT, refresh token, RBAC identity data. |
| Market Data | MQTT tick ingestion, validation, storage, and Kafka publication. |
| Order | Order creation, idempotency, lifecycle transitions, and events. |
| Portfolio | Consumes fills and updates holdings/cash snapshots. |
| Strategy | Strategy CRUD, backtests, generated signals. |
| Risk Engine | Risk score, VaR, volatility, drawdown, recommendations. |
| Surveillance | Consumes trading/risk/market events and creates alerts. |
| Notification | Creates notifications from surveillance events and records delivery attempts. |
| Audit | Stores searchable compliance-style audit logs and exports. |

## Event-Driven Flow

Orders publish lifecycle events. Portfolio consumes fills. Surveillance consumes order, market, risk, portfolio, and strategy events. Notifications consume surveillance alert events. Audit consumes key business and system events.

## Operational Layers

- Observability: health/readiness/metrics, Prometheus, Grafana dashboards, SLO docs.
- Reliability: retries, DLQ guidance, idempotency, graceful shutdown notes.
- Security: JWT/RBAC, threat model, RBAC matrix, gateway hardening, secrets guidance.
- Deployment readiness: Docker Compose first, optional Helm chart for Kubernetes discussion.
- Performance: safe local perf smoke, optional k6 scenarios, capacity planning guide.

## Senior Backend Signals

TradeOps demonstrates system design, event-driven architecture, production-readiness thinking, operational documentation, Go microservice implementation, cross-language service integration, and clear trade-off communication.
