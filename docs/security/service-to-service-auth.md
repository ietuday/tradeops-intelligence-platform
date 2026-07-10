# Service-To-Service Auth

Local service auth uses:

- `X-TradeOps-Service-Name`
- `X-TradeOps-Service-Token`

API Gateway adds these headers when proxying to backend services. Go services only trust normalized identity headers when the service token matches `SERVICE_AUTH_SHARED_SECRET`. Direct client requests with spoofed `X-TradeOps-*` headers are rejected unless they also present valid service auth.

This is a local/dev trust boundary. Production deployments should replace or supplement it with mTLS, service mesh identity, or workload identity while keeping the same normalized identity context.

