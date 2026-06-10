# Event Compatibility Rules

Event contract changes must be additive and backward-compatible by default.

## Backward-Compatible

- Add an optional field.
- Add metadata such as `eventId`, `eventVersion`, `tenantId`, `correlationId`, `traceparent`, `occurredAt`, or `source`.
- Add a new enum value only when all consumers tolerate unknown values.
- Relax validation.
- Add a new topic without changing an existing topic.
- Add a new schema version while continuing to publish the old shape during migration.

## Breaking

- Remove a field.
- Rename a field.
- Change a field type.
- Change field semantics while keeping the same name.
- Make an optional field required.
- Change topic meaning.
- Move all domain fields under `payload` before consumers support both flat and wrapped shapes.

## Review Questions

- Which services produce the event?
- Which services consume it?
- Do consumers ignore unknown fields?
- Does the event carry tenant context when tenant-owned?
- Is `correlationId` preserved for debugging and audit lookup?
- Is `traceparent` preserved where OpenTelemetry context is available?
- Are sample payloads and schemas updated together?
