from datetime import datetime, timezone
from statistics import NormalDist
from typing import Any

import numpy as np
import pandas as pd


class VarService:
    def calculate(self, snapshots: list[dict[str, Any]], confidence_level: float = 95, time_horizon_days: int = 1) -> dict[str, Any]:
        values = [float(item["total_value"]) for item in snapshots]
        if len(values) < 3:
            portfolio_value = values[-1] if values else 100000.0
            return {
                "value_at_risk": round(portfolio_value * 0.01, 4),
                "confidence_level": confidence_level,
                "time_horizon_days": time_horizon_days,
                "method": "fallback",
                "calculated_at": datetime.now(timezone.utc),
            }
        returns = pd.Series(values).pct_change().dropna()
        mean = float(returns.mean())
        std = float(returns.std())
        z = NormalDist().inv_cdf(1 - confidence_level / 100)
        portfolio_value = values[-1]
        var = abs(portfolio_value * (mean + z * std) * np.sqrt(time_horizon_days))
        return {
            "value_at_risk": round(float(var), 4),
            "confidence_level": confidence_level,
            "time_horizon_days": time_horizon_days,
            "method": "parametric",
            "calculated_at": datetime.now(timezone.utc),
        }
