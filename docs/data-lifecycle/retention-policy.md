# Data Retention Policy

TradeOps is a local portfolio platform, not a regulated production trading system. These retention periods are practical defaults for demos, interviews, and local operations. A real trading platform would need legal, compliance, privacy, and storage-cost review before adopting retention periods.

## Recommended Retention

| Domain | Why Retention Matters | Demo/Local Retention | Production Starting Point | Archive Candidate | Deletion Safety Notes |
| --- | --- | --- | --- | --- | --- |
| `market_ticks` | High-volume market data grows quickly and is mostly used for recent strategy/risk demos. | 30 days | 30-90 days hot, longer in warehouse if needed | Yes | Archive by `received_at`; never delete without confirming downstream strategy/risk needs. |
| `orders` | Orders are core business records and useful for auditability. | Retain indefinitely | Archive after 365 days, retain according to compliance policy | Archive only | Do not delete by default; preserve order history unless a formal retention policy exists. |
| `order_events` | Event history explains order lifecycle transitions and replay behavior. | 180 days | 365+ days or compliance-specific | Yes | Archive by `occurred_at`; deleting events can make historical order debugging harder. |
| `portfolio_snapshots` | Snapshots can grow over time but are useful for risk and demo history. | 180 days | 180-365 days hot, archive older snapshots | Yes | Archive by `created_at`; keep recent snapshots for risk demos. |
| `risk_scores` | Risk trends are useful for explainability and dashboards. | 180 days | 365 days or compliance-specific | Yes | Archive by `created_at`; do not delete active risk context unexpectedly. |
| `surveillance_alerts` | Alerts represent compliance-style records and investigations. | 365 days | 1-7 years depending on regulation | Archive only | Do not delete by default; preserve alert lifecycle history. |
| `notifications` | User notification history is useful but less durable than orders/audit. | 90 days | 90-365 days depending on policy | Yes | Archive by `created_at`; preserve unread or failed records until investigated. |
| `notification_delivery_attempts` | Delivery attempts explain webhook/email failures and retries. | 90 days | 90-180 days | Yes | Archive by `attempted_at`; keep enough data for delivery troubleshooting. |
| `audit_logs` | Audit records support compliance, incident review, and traceability. | 365 days minimum | 1-7 years depending on regulation | Archive only | Do not delete by default; deletion requires explicit compliance approval. |
| `audit_export_requests` | Export records are operational metadata for audit API use. | 90 days | 180-365 days | Yes | Archive by `created_at`; ensure exported data is handled safely. |
| Strategy/backtest tables | Backtests can be regenerated but are useful for demos and performance comparison. | 180 days for runs/signals, retain strategies | 180-365 days for runs/signals, retain strategies | Yes | Archive generated runs/signals before cleanup; avoid deleting strategy definitions casually. |
| DLQ messages | Failed messages need investigation and root-cause fixes. | Investigate within 7 days, archive after 30 days | Investigate within 7 days, archive after 30 days | Yes | Replay only after fixing the root cause; do not bulk replay unknown failures. |

## Policy Notes

- Archive before delete.
- Destructive cleanup should require explicit confirmation.
- Orders, surveillance alerts, and audit logs are intentionally excluded from default deletion scripts.
- Local archives should be treated as sensitive because they may contain user IDs, order details, webhook URLs, or audit metadata.
- Retention values are recommendations, not enforced service behavior.

