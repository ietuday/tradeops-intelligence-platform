# Event Schema Governance

TradeOps uses Kafka/Redpanda events across orders, portfolio, risk, surveillance, notifications, audit, and WebSocket streaming. Versioned event schemas make those contracts explicit so producers and consumers can evolve safely without needing a live schema registry in local demos.

## Current Approach

v2.5.0 uses lightweight JSON Schema files stored in the repository:

```text
schemas/events/
  common/
  market/
  orders/
  portfolio/
  strategy/
  risk/
  surveillance/
  notifications/
  audit/
```

Each schema is versioned in the filename, for example `order.created.v1.json`. The schema version describes the contract version, not the application release version.

This release does not add Redpanda Schema Registry or enforce schemas at runtime. Producers and consumers remain backward-compatible with existing flat event payloads.

v2.6.0 adds schemas for surveillance rule configuration events so rule threshold updates, enable actions, and disable actions are governed by the same repository-local contract process.

## Naming Convention

Use dot-separated event names:

```text
<domain>.<entity-or-action>.<lifecycle>
```

Examples:

- `order.created`
- `portfolio.updated`
- `surveillance.alert.created`
- `notification.retry_requested`
- `audit.log.created`

## Versioning Convention

- Start each event contract at `v1`.
- Use `eventVersion: "1.0"` in payloads where practical.
- Additive, compatible payload changes stay on the same major version.
- Breaking changes require a new schema file, a migration plan, and dual-consumer support during rollout.

## Compatibility Rules

Backward-compatible changes include:

- Adding an optional field.
- Adding metadata such as `eventVersion`, `tenantId`, `correlationId`, or `traceparent`.
- Relaxing validation.
- Adding a new enum value only when consumers tolerate unknown values.

Breaking changes include:

- Removing a field.
- Renaming a field.
- Changing a field type.
- Making an optional field required.
- Changing the meaning of a topic.

See [compatibility rules](compatibility-rules.md) for the full checklist.

## Adding A New Event Schema

1. Add a schema under `schemas/events/<domain>/<topic>.v1.json`.
2. Keep `additionalProperties: true` unless every consumer is known to tolerate strict validation.
3. Require only stable domain-critical fields.
4. Add or update a sample payload under `docs/examples/`.
5. Add the sample to `schemas/events/sample-mapping.json`.
6. Update [event catalog](event-catalog.md).
7. Run:

```bash
./scripts/validate-event-schemas.sh
```

## Handling Breaking Changes

Breaking changes should use a new major schema file and a transition period:

1. Publish old and new fields together when possible.
2. Update consumers to read both shapes.
3. Add a new schema version.
4. Update samples and docs.
5. Remove old fields only after every consumer no longer needs them.

## Future Option

The next step would be Redpanda/Kafka Schema Registry integration. That would allow runtime compatibility checks, producer-side validation, and registry-backed schema discovery. v2.5.0 intentionally stops short of that so local demos remain simple.
