# Scaling And High Availability

The chart supports:

- configurable replicas
- rolling updates
- HPA for stateless services
- PDBs for multi-replica critical workloads
- node selectors, tolerations, affinity, priority class, and topology spread constraints

Production values demonstrate multiple replicas for API Gateway, Identity, Order, Portfolio, Risk, Surveillance, Notification, Audit, and frontends. HPA requires CPU requests, which are included in application defaults.

If API Gateway WebSocket state remains pod-local, scaling can distribute connections across pods but does not move an active connection between pods. The existing Kafka-backed stream model should remain the source of shared event state.

