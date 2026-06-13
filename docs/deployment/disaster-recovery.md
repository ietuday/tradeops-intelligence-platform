# Disaster Recovery Guidance

Implemented in v3.0.0: deployment primitives and documentation hooks for backup-aware operation.

Recommended for production:

- PostgreSQL managed backups, PITR, restore drills, encryption at rest, TLS in transit
- Redis replication or documented recreation strategy
- Kafka/Redpanda topic replication, retention, and broker recovery runbooks
- Kubernetes object backups for namespaces, ConfigMaps, and release metadata
- Secret recovery through a managed secret store
- container image registry replication
- observability retention policies
- documented RPO and RTO targets

This release does not install a backup operator or automate multi-region recovery.

