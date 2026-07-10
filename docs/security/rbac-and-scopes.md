# RBAC And Scopes

The gateway normalizes token claims into roles and scopes, then applies route policy before proxying.

Default route policy:

- `GET /api/orders*`: `orders:read`
- `POST /api/orders`: `orders:write`
- order amend/cancel: `orders:write` or `orders:cancel`
- `GET /api/portfolio*`: `portfolio:read`
- `GET /api/audit*`: `admin`, `trading_admin`, or `audit:read`
- `/api/admin*` and `/internal*`: admin/internal role or service auth

Unknown protected routes require authentication. Health and readiness remain public. Auth errors use safe 401/403 responses and do not expose token contents.

