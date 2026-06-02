from datetime import datetime, timedelta, timezone

from app.services.anomaly_service import AnomalyService


def test_anomaly_detects_volume_spike():
    start = datetime(2026, 6, 1, tzinfo=timezone.utc)
    ticks = [
        {"symbol": "AAPL", "price": 100 + idx, "volume": 100, "event_time": start + timedelta(minutes=idx)}
        for idx in range(8)
    ]
    ticks.append({"symbol": "AAPL", "price": 109, "volume": 5000, "event_time": start + timedelta(minutes=9)})

    anomalies = AnomalyService().detect(ticks)

    assert any(item["type"] == "VOLUME_SPIKE" for item in anomalies)
