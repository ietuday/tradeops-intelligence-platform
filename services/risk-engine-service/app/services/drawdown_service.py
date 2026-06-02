from datetime import datetime, timezone
from typing import Any

import pandas as pd


class DrawdownService:
    def calculate(self, snapshots: list[dict[str, Any]]) -> dict[str, Any]:
        if not snapshots:
            return {"max_drawdown": 0.0, "current_drawdown": 0.0, "samples": 0, "calculated_at": datetime.now(timezone.utc)}
        values = pd.Series([float(item["total_value"]) for item in snapshots])
        peak = values.cummax()
        drawdowns = ((values - peak) / peak.replace(0, 1)) * 100
        return {
            "max_drawdown": round(abs(float(drawdowns.min())), 4),
            "current_drawdown": round(abs(float(drawdowns.iloc[-1])), 4),
            "samples": len(values),
            "calculated_at": datetime.now(timezone.utc),
        }
