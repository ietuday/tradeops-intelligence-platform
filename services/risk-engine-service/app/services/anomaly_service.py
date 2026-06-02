from datetime import datetime, timezone
from typing import Any

import pandas as pd


class AnomalyService:
    def detect(self, ticks: list[dict[str, Any]]) -> list[dict[str, Any]]:
        if not ticks:
            return []
        frame = pd.DataFrame(ticks)
        anomalies: list[dict[str, Any]] = []
        for symbol, group in frame.groupby("symbol"):
            ordered = group.sort_values("event_time").copy()
            ordered["price_change"] = ordered["price"].astype(float).pct_change().fillna(0)
            anomalies.extend(self._zscore_rows(symbol, ordered, "price_change", "PRICE_CHANGE"))
            anomalies.extend(self._zscore_rows(symbol, ordered, "volume", "VOLUME_SPIKE"))
        return anomalies[-20:]

    def _zscore_rows(self, symbol: str, frame: pd.DataFrame, column: str, anomaly_type: str) -> list[dict[str, Any]]:
        series = frame[column].astype(float)
        std = float(series.std())
        if len(series) < 3 or std == 0:
            return []
        mean = float(series.mean())
        found = []
        for row in frame.itertuples():
            value = float(getattr(row, column))
            z_score = abs((value - mean) / std)
            if z_score >= 2.5:
                found.append(
                    {
                        "symbol": symbol,
                        "type": anomaly_type,
                        "severity": "HIGH" if z_score >= 3 else "MEDIUM",
                        "value": value,
                        "z_score": round(float(z_score), 4),
                        "event_time": row.event_time if hasattr(row.event_time, "tzinfo") else datetime.now(timezone.utc),
                    }
                )
        return found
