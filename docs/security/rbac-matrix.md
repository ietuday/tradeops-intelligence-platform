# RBAC Matrix

This matrix documents the target role model for TradeOps and the current implementation posture. Services enforce JWT/RBAC where implemented, but this document is intentionally honest: it is a target reference for review and interview explanation, not a claim that every endpoint has identical enforcement depth.

## Roles

| Role | Intent |
| --- | --- |
| `trading_admin` | Full operational/admin role for local demos. |
| `trader` | Creates and manages own trading activity where supported. |
| `risk_manager` | Reviews risk and surveillance workflows, including alert resolution. |
| `analyst` | Reads market, portfolio, risk, surveillance, notification, and audit data for investigation. |
| `viewer` | Read-only access where supported. |

## Domain Access

| Domain/API | Read Access | Write Access | Lifecycle Actions | Export Actions | Admin Actions |
| --- | --- | --- | --- | --- | --- |
| Auth | All authenticated users for own profile | Anonymous login/register; user updates where supported | Refresh/logout own session | None | `trading_admin` for admin/user config if added |
| Market data | `trading_admin`, `trader`, `risk_manager`, `analyst`, `viewer` | `trading_admin` for simulated/admin feeds | None | `trading_admin`, `analyst` if export added | `trading_admin` |
| Orders | `trading_admin`, `trader`, `risk_manager`, `analyst` | `trading_admin`, `trader` for own orders | `trading_admin`, `trader` can create/cancel own orders if supported | `trading_admin`, `analyst` if export added | `trading_admin` |
| Portfolio | `trading_admin`, `trader` for own data, `risk_manager`, `analyst`, `viewer` | System/event updates | Recalculation/import if added: `trading_admin` | `trading_admin`, `analyst` if export added | `trading_admin` |
| Strategies | `trading_admin`, `trader`, `risk_manager`, `analyst` | `trading_admin`, `trader` | Backtest/generate signals: `trading_admin`, `trader` | `trading_admin`, `analyst` if export added | `trading_admin` |
| Risk | `trading_admin`, `risk_manager`, `analyst`, `viewer` | System/service generated | Risk review actions: `trading_admin`, `risk_manager` | `trading_admin`, `risk_manager`, `analyst` | `trading_admin` |
| Surveillance | `trading_admin`, `risk_manager`, `analyst`, `viewer` | System/service generated | Acknowledge: `trading_admin`, `risk_manager`; resolve/dismiss: `trading_admin`, `risk_manager` | `trading_admin`, `risk_manager`, `analyst` if export added | `trading_admin` |
| Notifications | Authenticated users for own notifications; `trading_admin` for admin views if added | Preferences: notification owner | Mark read/retry own notifications; admin retry if added | None | `trading_admin` |
| Audit | `trading_admin`, `risk_manager`, `analyst` | System/event generated | None | `trading_admin`, `risk_manager` | `trading_admin` |
| Admin/config | `trading_admin` | `trading_admin` | `trading_admin` | `trading_admin` | `trading_admin` |

## Documented Target RBAC Vs Current Implementation

- The target posture is least privilege by domain and action.
- Current services implement JWT/RBAC at different depths; this is acceptable for the local portfolio scope but should be normalized before production.
- The API Gateway forwards authorization headers and keeps a single external routing boundary, but backend services remain responsible for final authorization decisions.
- Only `trading_admin` and `risk_manager` should resolve surveillance alerts.
- Only `trading_admin` and `risk_manager` should export audit logs.
- `viewer` should be read-only wherever the service supports that role.
- `trader` should create/cancel only their own orders where ownership is modeled.

## Review Checklist

- Confirm each protected endpoint has an explicit role list.
- Confirm lifecycle actions are stricter than read actions.
- Confirm export endpoints are treated as sensitive data access.
- Confirm tokens used in demos match the expected role.
