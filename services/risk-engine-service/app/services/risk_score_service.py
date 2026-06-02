from datetime import datetime, timezone
from typing import Any


class RiskScoreService:
    def calculate(
        self,
        portfolio: dict[str, Any],
        holdings: list[dict[str, Any]],
        latest_prices: dict[str, float],
        volatility: dict[str, Any],
        drawdown: dict[str, Any],
        recent_anomaly_count: int,
    ) -> dict[str, Any]:
        cash = float(portfolio.get("cash_balance", 0))
        holding_values = [
            float(item["quantity"]) * float(latest_prices.get(item["symbol"], item["average_buy_price"]))
            for item in holdings
        ]
        holdings_value = sum(holding_values)
        total_value = max(cash + holdings_value, 1)
        concentration = max(holding_values) / total_value if holding_values else 0
        avg_volatility = self._avg_volatility(volatility)
        low_cash = 1 - min(cash / total_value, 1)
        unrealized_loss = self._unrealized_loss(holdings, latest_prices, total_value)
        anomaly_factor = min(recent_anomaly_count / 5, 1)
        drawdown_factor = min(float(drawdown.get("max_drawdown", 0)) / 30, 1)
        volatility_factor = min(avg_volatility / 80, 1)

        factors = {
            "portfolioConcentration": round(concentration * 100, 4),
            "symbolVolatility": round(volatility_factor * 100, 4),
            "portfolioDrawdown": round(drawdown_factor * 100, 4),
            "lowCashBuffer": round(low_cash * 100, 4),
            "recentAnomalyCount": round(anomaly_factor * 100, 4),
            "unrealizedLoss": round(unrealized_loss * 100, 4),
        }
        score = (
            factors["portfolioConcentration"] * 0.20
            + factors["symbolVolatility"] * 0.20
            + factors["portfolioDrawdown"] * 0.20
            + factors["lowCashBuffer"] * 0.15
            + factors["recentAnomalyCount"] * 0.15
            + factors["unrealizedLoss"] * 0.10
        )
        return {"score": round(min(max(score, 0), 100), 4), "level": risk_level(score), "factors": factors, "calculated_at": datetime.now(timezone.utc)}

    def _avg_volatility(self, volatility: dict[str, Any]) -> float:
        items = volatility.get("symbols", [])
        if not items:
            return 0.0
        return sum(float(item["volatility"]) for item in items) / len(items)

    def _unrealized_loss(self, holdings: list[dict[str, Any]], latest_prices: dict[str, float], total_value: float) -> float:
        loss = 0.0
        for item in holdings:
            qty = float(item["quantity"])
            avg = float(item["average_buy_price"])
            current = float(latest_prices.get(item["symbol"], avg))
            loss += max((avg - current) * qty, 0)
        return min(loss / total_value, 1)


def risk_level(score: float) -> str:
    if score <= 30:
        return "LOW"
    if score <= 60:
        return "MEDIUM"
    if score <= 80:
        return "HIGH"
    return "CRITICAL"
