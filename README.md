# TradeOps Intelligence Platform

TradeOps Intelligence Platform is an enterprise-grade local trading, risk, AI analytics, and observability platform.

The platform is designed to run completely on a local machine without any cloud account.

## v0.1.0 Scope

This release creates the platform foundation.

### Included

- Monorepo structure
- Node.js TypeScript API Gateway
- Angular shell placeholder
- React trading dashboard placeholder
- PostgreSQL
- Redis
- Eclipse Mosquitto MQTT broker
- Redpanda Kafka-compatible event bus
- Prometheus
- Grafana
- Docker Compose setup
- Makefile
- Smoke test script
- Release notes

### Not Included Yet

- Authentication
- Market data service
- Order service
- Portfolio service
- Strategy service
- Risk engine
- Notification service
- AI assistant bot
- Local Kubernetes deployment

## Local Prerequisites

- Docker
- Docker Compose
- Make
- Node.js, optional for local non-Docker development

## Local URLs

| Component | URL |
|---|---|
| API Gateway | http://localhost:8080 |
| API Gateway Health | http://localhost:8080/health |
| API Gateway Ready | http://localhost:8080/ready |
| API Gateway Metrics | http://localhost:8080/metrics |
| Angular Shell | http://localhost:4200 |
| React Trading Dashboard | http://localhost:4300 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| Mosquitto MQTT | localhost:1883 |
| Redpanda Kafka | localhost:9092 |

### Local Docker environment

Create a local Docker environment file before starting the stack:

```bash
cp infrastructure/docker/.env.example infrastructure/docker/.env

## Commands

```bash
make dev-up
make ps
make smoke
make logs
make dev-down