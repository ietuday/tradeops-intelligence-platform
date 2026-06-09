# Architecture Summary

## High-Level Architecture

```mermaid
flowchart LR
  client[Client] --> gateway[API Gateway]
  gateway --> identity[Identity]
  gateway --> market[Market Data]
  gateway --> orders[Order]
  gateway --> portfolio[Portfolio]
  gateway --> strategy[Strategy]
  gateway --> risk[Risk]
  gateway --> surveillance[Surveillance]
  gateway --> notification[Notification]
  gateway --> audit[Audit]

  identity --> postgres[(PostgreSQL)]
  orders --> postgres
  portfolio --> postgres
  surveillance --> postgres
  notification --> postgres
  audit --> postgres
  identity --> redis[(Redis)]
  mqtt[(Mosquitto/MQTT)] --> market
  market --> kafka[(Redpanda/Kafka)]
  orders --> kafka
  portfolio --> kafka
  strategy --> kafka
  risk --> kafka
  surveillance --> kafka
  notification --> kafka
  audit --> kafka
```

The gateway provides the external HTTP boundary. Services own domain logic and data while Kafka handles asynchronous workflows.

## Event Flow

```mermaid
flowchart LR
  market[market ticks] --> surveillance
  orders[order events] --> portfolio
  orders --> surveillance
  portfolio[portfolio.updated] --> surveillance
  risk[risk.score.updated] --> surveillance
  strategy[strategy.signal.generated] --> surveillance
  surveillance[surveillance.alert.*] --> notification
  orders --> audit
  surveillance --> audit
  notification --> audit
```

Events let services evolve independently and make replay/DLQ workflows possible.

## Request Flow

```mermaid
sequenceDiagram
  participant C as Client
  participant G as API Gateway
  participant S as Service
  participant DB as PostgreSQL
  C->>G: HTTP request + JWT + X-Correlation-ID
  G->>S: Proxied request
  S->>S: JWT/RBAC validation
  S->>DB: Query/update
  S-->>G: JSON response
  G-->>C: JSON response + correlation ID
```

Synchronous APIs are used for commands and queries; events carry side effects and integration signals.

## Observability Flow

```mermaid
flowchart LR
  services[Services /metrics] --> prometheus[Prometheus]
  prometheus --> grafana[Grafana]
  services --> logs[Structured logs]
  gateway[X-Correlation-ID] --> services
  services --> events[Events with correlationId]
  events --> audit[Audit logs]
```

Metrics, logs, events, and audit records share correlation identifiers for lightweight tracing.

## Deployment Flow

```mermaid
flowchart LR
  compose[Docker Compose] --> local[Full local platform]
  helm[Helm chart] --> k8s[Kubernetes application manifests]
  k8s --> managed[Managed PostgreSQL/Redis/Kafka in production]
```

Docker Compose is the primary runnable demo. Helm is an optional deployment-readiness layer for explaining Kubernetes packaging.
