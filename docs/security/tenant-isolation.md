# Tenant Isolation

Tenant identity is authoritative from the validated access token and normalized gateway headers.

Rules:

- API Gateway requires a tenant claim for user JWTs by default.
- Backend services prefer `X-TradeOps-Tenant-Id` only when service auth proves the caller is the gateway or another trusted service.
- Order and Portfolio handlers build their service user context from the trusted tenant, and repositories continue to include tenant filters.
- Client-supplied tenant headers are stripped at the gateway boundary.

Cross-tenant administrative access requires explicit admin role/scope and should be audited.

