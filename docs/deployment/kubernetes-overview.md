# Kubernetes Deployment Overview

TradeOps v3.0.0 adds a cloud-neutral Kubernetes blueprint centered on the Helm chart in `deployments/helm/tradeops`.

```mermaid
flowchart LR
  User[User] --> Ingress[Ingress]
  Ingress --> Angular[Angular Shell]
  Ingress --> Dashboard[React Dashboard]
  Ingress --> Gateway[API Gateway]
  Gateway --> Identity[Identity]
  Gateway --> Orders[Order]
  Gateway --> Portfolio[Portfolio]
  Gateway --> Strategy[Strategy]
  Gateway --> Risk[Risk Engine]
  Gateway --> Surveillance[Surveillance]
  Gateway --> Notifications[Notification]
  Gateway --> Audit[Audit]
  Identity --> Postgres[(PostgreSQL)]
  Orders --> Postgres
  Portfolio --> Postgres
  Risk --> Postgres
  Surveillance --> Postgres
  Notifications --> Postgres
  Audit --> Postgres
  Identity --> Redis[(Redis)]
  Orders --> Kafka[(Kafka/Redpanda)]
  Portfolio --> Kafka
  Risk --> Kafka
  Surveillance --> Kafka
  Notifications --> Kafka
  Audit --> Kafka
  Market[Market Data] --> MQTT[(MQTT)]
  Market --> Kafka
  Prometheus[Prometheus] --> Gateway
  Grafana[Grafana] --> Prometheus
  Gateway --> OTel[OpenTelemetry Collector]
  OTel --> Jaeger[Jaeger or OTLP Backend]
```

Local values can deploy demo PostgreSQL, Redis, Redpanda, and Mosquitto. Staging and production values prefer externally managed dependencies.

Known production ownership remains outside the chart: sizing, DNS, TLS issuance, credential lifecycle, backup policies, image registry replication, and incident response.

## Request Flow

```mermaid
flowchart LR
  User[User] --> Ingress[Ingress]
  Ingress --> Frontend[Angular or React Frontend]
  Frontend --> Gateway[API Gateway]
  Ingress --> Gateway
  Gateway --> Backend[Backend Service]
  Backend --> Database[(Database)]
  Backend --> Cache[(Redis)]
  Backend --> Broker[(Kafka/Redpanda)]
```

## Deployment Flow

```mermaid
flowchart LR
  Developer[Developer or CI] --> Registry[Container Registry]
  Developer --> Helm[Helm Chart]
  Registry --> Kubernetes[Kubernetes Cluster]
  Helm --> Kubernetes
  Kubernetes --> Migration[Migration Job]
  Migration --> Rollout[Application Rollout]
  Rollout --> Smoke[Smoke Validation]
```
