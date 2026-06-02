from datetime import datetime, timezone
from typing import Any

import pandas as pd


class VolatilityService:
    def calculate(self, ticks: list[dict[str, Any]]) -> dict[str, Any]:
        if not ticks:
            return {"symbols": [], "calculated_at": datetime.now(timezone.utc)}
        frame = pd.DataFrame(ticks)
        result = []
        for symbol, group in frame.groupby("symbol"):
            prices = group.sort_values("event_time")["price"].astype(float)
            returns = prices.pct_change().dropna()
            volatility = float(returns.std() * (252 ** 0.5) * 100) if len(returns) > 1 else 0.0
            result.append({"symbol": symbol, "volatility": round(volatility, 4), "samples": int(len(prices))})
        return {"symbols": result, "calculated_at": datetime.now(timezone.utc)}
