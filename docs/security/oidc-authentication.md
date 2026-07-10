# OIDC Authentication

TradeOps v3.5.0 validates API Gateway bearer tokens with OIDC-style JWKS verification.

Local development uses Identity Service as a mock issuer:

- Discovery: `GET /.well-known/openid-configuration`
- JWKS: `GET /.well-known/jwks.json`
- Login/token: `POST /api/v1/auth/login` and `POST /api/v1/auth/token`

Access tokens are signed with RS256 when `IDENTITY_OIDC_ENABLED=true`. If `IDENTITY_JWT_PRIVATE_KEY_PATH` is unset, Identity Service generates an ephemeral local RSA key and publishes the matching public key through JWKS. Do not use ephemeral keys in production because tokens are invalidated on restart.

API Gateway validates issuer, audience, expiration, not-before, `kid`, and an algorithm allowlist before forwarding trusted identity headers. It strips incoming `X-TradeOps-*` identity headers from client requests.

Production IdP configuration is via `OIDC_ISSUER_URL`, `OIDC_JWKS_URL`, `OIDC_AUDIENCE`, `OIDC_ALLOWED_ALGORITHMS`, `OIDC_JWKS_CACHE_TTL`, and `AUTH_CLOCK_SKEW`.

