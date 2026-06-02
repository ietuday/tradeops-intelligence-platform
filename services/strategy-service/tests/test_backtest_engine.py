from datetime import datetime, timedelta, timezone

import pytest

from app.services.backtest_engine import BacktestEngine, BacktestValidationError


def prices(values):
    start = datetime(2026, 6, 1, tzinfo=timezone.utc)
    return [{"price": value, "event_time": start + timedelta(minutes=index)} for index, value in enumerate(values)]


def test_moving_average_backtest_generates_performance_and_signal():
    result = BacktestEngine().run(
        "MOVING_AVERAGE_CROSSOVER",
        {"shortWindow": 2, "longWindow": 3},
        prices([100, 99, 98, 101, 103, 104]),
        10000,
    )

    assert result.performance["total_trades"] >= 1
    assert result.signals[0]["signal"] in {"BUY", "SELL", "HOLD"}


def test_backtest_rejects_insufficient_data():
    with pytest.raises(BacktestValidationError, match="Not enough market data"):
        BacktestEngine().run("RSI", {"period": 14}, prices([100, 101]), 10000)
