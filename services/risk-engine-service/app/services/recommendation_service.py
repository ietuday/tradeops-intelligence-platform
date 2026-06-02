from typing import Any


class RecommendationService:
    def generate(
        self,
        score: dict[str, Any],
        volatility: dict[str, Any],
        drawdown: dict[str, Any],
        var: dict[str, Any],
        anomalies: list[dict[str, Any]],
    ) -> list[dict[str, Any]]:
        recommendations: list[dict[str, Any]] = []
        factors = score.get("factors", {})
        if score["score"] > 60 or factors.get("portfolioConcentration", 0) > 40:
            recommendations.append({"type": "REDUCE_EXPOSURE", "message": "Reduce exposure in concentrated positions.", "severity": score["level"], "context": factors})
        high_vol = [item for item in volatility.get("symbols", []) if item["volatility"] > 30]
        for item in high_vol[:3]:
            recommendations.append({"type": "REVIEW_HIGH_VOLATILITY_SYMBOL", "message": f"Review high-volatility symbol {item['symbol']}.", "severity": "MEDIUM", "context": item})
        if factors.get("lowCashBuffer", 0) > 70:
            recommendations.append({"type": "INCREASE_CASH_BUFFER", "message": "Increase cash buffer to reduce liquidity risk.", "severity": "MEDIUM", "context": factors})
        if drawdown.get("max_drawdown", 0) > 10 or var.get("value_at_risk", 0) > 2000:
            recommendations.append({"type": "ADD_STOP_LOSS", "message": "Add or review stop-loss levels for active holdings.", "severity": "HIGH", "context": {"drawdown": drawdown, "var": var}})
        if anomalies:
            recommendations.append({"type": "INVESTIGATE_ANOMALY", "message": "Investigate recent market anomaly before increasing exposure.", "severity": "HIGH", "context": {"anomalyCount": len(anomalies)}})
        if not recommendations:
            recommendations.append({"type": "MONITOR_RISK", "message": "Portfolio risk is within expected local thresholds.", "severity": "LOW", "context": factors})
        return recommendations
