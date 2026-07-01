from datetime import datetime, timedelta, timezone
from decimal import Decimal
from typing import Any

from sqlalchemy import text
from sqlalchemy.orm import Session

from app import models


class RiskRepository:
    def __init__(self, db: Session):
        self.db = db

    def get_portfolio(self, user_id: str) -> dict[str, Any]:
        row = self.db.execute(
            text(
                """
                SELECT p.id::text AS portfolio_id,
                       COALESCE(cb.cash_balance::float8, 100000) AS cash_balance,
                       COALESCE(cb.realized_pnl::float8, 0) AS realized_pnl
                FROM portfolios p
                LEFT JOIN cash_balances cb ON cb.portfolio_id = p.id
                WHERE p.user_id = :user_id
                """
            ),
            {"user_id": user_id},
        ).first()
        if not row:
            return {"portfolio_id": None, "cash_balance": 100000.0, "realized_pnl": 0.0}
        return {"portfolio_id": row.portfolio_id, "cash_balance": row.cash_balance, "realized_pnl": row.realized_pnl}

    def get_holdings(self, user_id: str) -> list[dict[str, Any]]:
        rows = self.db.execute(
            text(
                """
                SELECT symbol, quantity::float8 AS quantity, average_buy_price::float8 AS average_buy_price
                FROM portfolio_holdings
                WHERE user_id = :user_id
                ORDER BY symbol
                """
            ),
            {"user_id": user_id},
        )
        return [dict(row._mapping) for row in rows]

    def get_snapshots(self, user_id: str, limit: int = 100) -> list[dict[str, Any]]:
        rows = self.db.execute(
            text(
                """
                SELECT total_value::float8 AS total_value, cash_balance::float8 AS cash_balance, created_at
                FROM portfolio_snapshots
                WHERE user_id = :user_id
                ORDER BY created_at DESC
                LIMIT :limit
                """
            ),
            {"user_id": user_id, "limit": limit},
        )
        return list(reversed([dict(row._mapping) for row in rows]))

    def get_market_ticks(self, symbols: list[str], hours: int = 24) -> list[dict[str, Any]]:
        if not symbols:
            symbols = ["AAPL"]
        rows = self.db.execute(
            text(
                """
                SELECT symbol, price::float8 AS price, volume::float8 AS volume, event_time
                FROM market_ticks
                WHERE symbol = ANY(:symbols)
                  AND event_time >= :start_time
                ORDER BY symbol, event_time ASC
                """
            ),
            {"symbols": symbols, "start_time": datetime.now(timezone.utc) - timedelta(hours=hours)},
        )
        return [dict(row._mapping) for row in rows]

    def get_latest_prices(self, symbols: list[str]) -> dict[str, float]:
        if not symbols:
            return {}
        rows = self.db.execute(
            text(
                """
                SELECT DISTINCT ON (symbol) symbol, price::float8 AS price
                FROM market_ticks
                WHERE symbol = ANY(:symbols)
                ORDER BY symbol, event_time DESC
                """
            ),
            {"symbols": symbols},
        )
        return {row.symbol: row.price for row in rows}

    def count_recent_anomalies(self, user_id: str, hours: int = 24) -> int:
        row = self.db.execute(
            text(
                """
                SELECT COUNT(*) AS count
                FROM risk_anomalies
                WHERE user_id = :user_id
                  AND created_at >= :start_time
                """
            ),
            {"user_id": user_id, "start_time": datetime.now(timezone.utc) - timedelta(hours=hours)},
        ).first()
        return int(row.count if row else 0)

    def save_score(self, user_id: str, score: float, level: str, factors: dict[str, float]) -> models.RiskScore:
        risk_score = models.RiskScore(user_id=user_id, score=score, level=level, factors=factors)
        self.db.add(risk_score)
        if level in {"HIGH", "CRITICAL"}:
            self.db.add(models.RiskEvent(user_id=user_id, event_type="risk.breached", level=level, payload={"score": score, "factors": factors}))
        self.db.commit()
        self.db.refresh(risk_score)
        return risk_score

    def save_recommendations(self, user_id: str, recommendations: list[dict[str, Any]]) -> list[models.RiskRecommendation]:
        saved = [
            models.RiskRecommendation(
                user_id=user_id,
                recommendation_type=item["type"],
                message=item["message"],
                severity=item["severity"],
                context=item.get("context", {}),
            )
            for item in recommendations
        ]
        self.db.add_all(saved)
        self.db.commit()
        for item in saved:
            self.db.refresh(item)
        return saved

    def save_anomalies(self, user_id: str, anomalies: list[dict[str, Any]]) -> list[models.RiskAnomaly]:
        saved = [
            models.RiskAnomaly(
                user_id=user_id,
                symbol=item["symbol"],
                anomaly_type=item["type"],
                severity=item["severity"],
                value=item["value"],
                z_score=item["z_score"],
                event_time=item["event_time"],
            )
            for item in anomalies
        ]
        self.db.add_all(saved)
        self.db.commit()
        for item in saved:
            self.db.refresh(item)
        return saved

    def latest_recommendations(self, user_id: str) -> list[models.RiskRecommendation]:
        return (
            self.db.query(models.RiskRecommendation)
            .filter(models.RiskRecommendation.user_id == user_id)
            .order_by(models.RiskRecommendation.created_at.desc())
            .limit(20)
            .all()
        )

    def latest_anomalies(self, user_id: str) -> list[models.RiskAnomaly]:
        return (
            self.db.query(models.RiskAnomaly)
            .filter(models.RiskAnomaly.user_id == user_id)
            .order_by(models.RiskAnomaly.created_at.desc())
            .limit(50)
            .all()
        )

    def get_pre_trade_policy(self, tenant_id: str) -> dict[str, Any]:
        row = self.db.execute(
            text(
                """
                SELECT id::text, tenant_id, enabled, version,
                       max_order_quantity, max_order_notional,
                       max_daily_user_notional, max_daily_tenant_notional,
                       max_open_orders, allowed_symbols, restricted_symbols,
                       currency, risk_timezone
                FROM pre_trade_risk_policies
                WHERE enabled = TRUE
                  AND (tenant_id = :tenant_id OR tenant_id IS NULL)
                  AND effective_from <= now()
                  AND (effective_until IS NULL OR effective_until > now())
                ORDER BY CASE WHEN tenant_id = :tenant_id THEN 0 ELSE 1 END, effective_from DESC
                LIMIT 1
                """
            ),
            {"tenant_id": tenant_id},
        ).first()
        if not row:
            raise LookupError("No active pre-trade risk policy is configured.")
        return dict(row._mapping)

    def approved_notional_for_day(self, tenant_id: str, user_id: str, day_start: datetime, day_end: datetime) -> tuple[Decimal, Decimal]:
        row = self.db.execute(
            text(
                """
                SELECT
                  COALESCE(SUM(estimated_notional) FILTER (WHERE user_id = :user_id), 0) AS user_notional,
                  COALESCE(SUM(estimated_notional), 0) AS tenant_notional
                FROM order_risk_decisions
                WHERE tenant_id = :tenant_id
                  AND approved = TRUE
                  AND decision = 'APPROVED'
                  AND evaluated_at >= :day_start
                  AND evaluated_at < :day_end
                """
            ),
            {"tenant_id": tenant_id, "user_id": user_id, "day_start": day_start, "day_end": day_end},
        ).first()
        if not row:
            return Decimal("0"), Decimal("0")
        return Decimal(str(row.user_notional)), Decimal(str(row.tenant_notional))
