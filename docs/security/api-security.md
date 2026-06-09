# API Security

The API Gateway is the practical external security boundary for the local platform. Backend services still validate JWTs and enforce role checks where implemented.

## Controls

| Control | Current Posture |
| --- | --- |
| JWT validation | Services validate JWTs with the shared local signing secret. |
| RBAC enforcement | Implemented where service APIs require roles; see [RBAC matrix](rbac-matrix.md). |
| Correlation ID | Gateway accepts or generates `X-Correlation-ID` and forwards it downstream. |
| Request body size limit | Gateway JSON parser uses `REQUEST_BODY_LIMIT`, default `1mb`. Oversized JSON returns `413`. |
| Rate limiting | Gateway in-memory limiter uses `RATE_LIMIT_WINDOW_MS` and `RATE_LIMIT_MAX_REQUESTS`, default `60000`/`300`. |
| Security headers | Helmet is enabled and `x-powered-by` is disabled. |
| CORS | `CORS_ORIGIN` can restrict allowed origins. Local default allows `http://localhost:4200,http://localhost:4300` in Compose. |
| Proxy timeout handling | Proxy utilities return `504` for upstream timeout and `502` for upstream failures. |
| Error shape | Security and proxy errors return `{ error: { code, message, correlationId } }`. |
| Idempotency | Order creation supports `Idempotency-Key` to reduce duplicate side effects. |
| Audit export protection | Audit export should remain restricted to elevated roles. |
| Webhook delivery safety | Notification webhook delivery uses timeout/retry and records delivery attempts. |

## Environment Variables

```text
CORS_ORIGIN=http://localhost:4200,http://localhost:4300
REQUEST_BODY_LIMIT=1mb
RATE_LIMIT_WINDOW_MS=60000
RATE_LIMIT_MAX_REQUESTS=300
PROXY_TIMEOUT_MS=10000
```

## Input Validation Practices

- Validate request payloads at service boundaries.
- Keep route allowlists explicit in proxy routers.
- Treat event payloads as untrusted input and fail gracefully to retries/DLQ.
- Keep lifecycle actions stricter than read operations.

## Known Limitations

- In-memory rate limiting is per gateway process and resets on restart.
- Local Compose does not provide TLS termination or WAF protection.
- CORS is local-demo friendly unless explicitly configured for production.
- There is no OAuth/OIDC provider or mTLS in this release.
