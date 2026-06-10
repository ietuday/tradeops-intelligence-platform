# Event Contract Checklist

Use this checklist before adding or changing a Kafka/Redpanda event:

- [ ] Event name follows the dot-separated naming convention.
- [ ] Schema file added under `schemas/events/`.
- [ ] Schema filename includes a version, for example `.v1.json`.
- [ ] Required fields are limited to stable domain-critical fields.
- [ ] `additionalProperties: true` is used unless every consumer is ready for strict validation.
- [ ] Sample payload added under `docs/examples/`.
- [ ] Sample mapping added to `schemas/events/sample-mapping.json`.
- [ ] Producer emits `eventType` where practical.
- [ ] Producer emits `eventVersion` where practical.
- [ ] `tenantId` is included for tenant-owned events.
- [ ] `correlationId` is included or preserved.
- [ ] `traceparent` is included when available.
- [ ] Consumer tolerant parsing was reviewed.
- [ ] DLQ behavior was reviewed for malformed payloads.
- [ ] Docs and event catalog were updated.
- [ ] Tests or validation scripts were updated.
- [ ] Backward compatibility was checked.
