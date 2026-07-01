import uuid
from datetime import datetime, time, timedelta, timezone
from decimal import Decimal
from zoneinfo import ZoneInfo

from app.repositories.risk_repository import RiskRepository
from app.schemas import PreTradeRiskDecision, PreTradeRiskRequest


class PreTradeRiskService:
    def evaluate(self, repository: RiskRepository, payload: PreTradeRiskRequest) -> PreTradeRiskDecision:
        evaluated_at = datetime.now(timezone.utc)
        policy = repository.get_pre_trade_policy(payload.tenantId)
        evaluated_limits = self._limits(policy)

        reason = self._basic_rejection(payload, policy)
        if not reason:
            day_start, day_end = self._risk_day(policy["risk_timezone"], evaluated_at)
            user_notional, tenant_notional = repository.approved_notional_for_day(payload.tenantId, payload.userId, day_start, day_end)
            if user_notional + payload.estimatedNotional > Decimal(str(policy["max_daily_user_notional"])):
                reason = ("DAILY_USER_NOTIONAL_EXCEEDED", "Order exceeds maximum daily user notional")
            elif tenant_notional + payload.estimatedNotional > Decimal(str(policy["max_daily_tenant_notional"])):
                reason = ("DAILY_TENANT_NOTIONAL_EXCEEDED", "Order exceeds maximum daily tenant notional")

        if reason:
            approved = False
            decision = "REJECTED"
            reason_code, reason_message = reason
        else:
            approved = True
            decision = "APPROVED"
            reason_code = "RISK_OK"
            reason_message = "All pre-trade checks passed"

        return PreTradeRiskDecision(
            decisionId=str(uuid.uuid4()),
            approved=approved,
            decision=decision,
            reasonCode=reason_code,
            reasonMessage=reason_message,
            evaluatedLimits=evaluated_limits,
            evaluatedAt=evaluated_at,
            policyId=policy["id"],
            policyVersion=policy["version"],
        )

    def _basic_rejection(self, payload: PreTradeRiskRequest, policy: dict) -> tuple[str, str] | None:
        symbol = payload.symbol.strip().upper()
        restricted = {item.upper() for item in policy["restricted_symbols"] or []}
        allowed = {item.upper() for item in policy["allowed_symbols"] or []}
        if symbol in restricted:
            return "SYMBOL_RESTRICTED", "Symbol is restricted by pre-trade risk policy"
        if allowed and symbol not in allowed:
            return "SYMBOL_NOT_ALLOWED", "Symbol is not allowed by pre-trade risk policy"
        if payload.quantity > Decimal(str(policy["max_order_quantity"])):
            return "MAX_ORDER_QUANTITY_EXCEEDED", "Order exceeds maximum order quantity"
        if payload.estimatedNotional <= 0 or payload.estimatedPrice <= 0:
            return "INVALID_ESTIMATED_NOTIONAL", "Estimated notional is invalid"
        if payload.estimatedNotional > Decimal(str(policy["max_order_notional"])):
            return "MAX_ORDER_NOTIONAL_EXCEEDED", "Order exceeds maximum order notional"
        return None

    def _limits(self, policy: dict) -> dict[str, str]:
        return {
            "maxOrderQuantity": str(policy["max_order_quantity"]),
            "maxOrderNotional": str(policy["max_order_notional"]),
            "maxDailyUserNotional": str(policy["max_daily_user_notional"]),
            "maxDailyTenantNotional": str(policy["max_daily_tenant_notional"]),
        }

    def _risk_day(self, timezone_name: str, now: datetime) -> tuple[datetime, datetime]:
        location = ZoneInfo(timezone_name or "UTC")
        local_now = now.astimezone(location)
        local_start = datetime.combine(local_now.date(), time.min, tzinfo=location)
        local_end = local_start + timedelta(days=1)
        return local_start.astimezone(timezone.utc), local_end.astimezone(timezone.utc)
