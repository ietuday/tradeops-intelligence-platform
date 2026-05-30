```markdown
# Changelog

All notable changes to TradeOps Intelligence Platform will be documented in this file.

## [v0.1.0] - Platform Foundation

### Added

- Initial monorepo structure
- API Gateway foundation
- Docker Compose foundation
- PostgreSQL
- Redis
- Mosquitto MQTT
- Redpanda Kafka-compatible event bus
- Prometheus
- Grafana
- Angular shell placeholder
- React trading dashboard placeholder
- Smoke test script
- Release notes
Step 8: Add Release Notes

File:

docs/release-notes/v0.1.0.md

Content:

# v0.1.0 - Platform Foundation

## Added

- Initial monorepo structure
- API Gateway service foundation
- Angular shell placeholder
- React trading dashboard placeholder
- Docker Compose setup
- PostgreSQL container
- Redis container
- Mosquitto MQTT broker
- Redpanda Kafka-compatible event bus
- Prometheus metrics stack
- Grafana dashboard stack
- Makefile commands
- Smoke test script

## Infrastructure

- PostgreSQL exposed on port `5432`
- Redis exposed on port `6379`
- Mosquitto exposed on port `1883`
- Redpanda Kafka exposed on port `9092`
- Prometheus exposed on port `9090`
- Grafana exposed on port `3000`
- API Gateway exposed on port `8080`

## Testing

- Smoke test validates API Gateway `/health`, `/ready`, and `/metrics`

## Known Limitations

- Authentication is not implemented yet
- Market data pipeline is not implemented yet
- Trading order flow is not implemented yet
- AI assistant is not implemented yet
- Kubernetes deployment is planned for a later release

## Next Release

v0.2.0 will add Identity and RBAC.