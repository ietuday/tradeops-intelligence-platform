from dataclasses import dataclass
from datetime import datetime
from typing import Any

import numpy as np
import pandas as pd


class BacktestValidationError(ValueError):
    pass


@dataclass(frozen=True)
class BacktestResult:
    performance: dict[str, Any]
    signals: list[dict[str, Any]]


class BacktestEngine:
    def run(
        self,
        strategy_type: str,
        parameters: dict[str, Any],
        prices: list[dict[str, Any]],
        initial_capital: float,
    ) -> BacktestResult:
        frame = pd.DataFrame(prices)
        if frame.empty:
            raise BacktestValidationError("Not enough market data for this strategy and time range.")
        frame["event_time"] = pd.to_datetime(frame["event_time"], utc=True)
        frame["price"] = frame["price"].astype(float)

        if strategy_type == "MOVING_AVERAGE_CROSSOVER":
            signals = self._moving_average_signals(frame, parameters)
        elif strategy_type == "RSI":
            signals = self._rsi_signals(frame, parameters)
        elif strategy_type == "VOLATILITY_BREAKOUT":
            signals = self._volatility_breakout_signals(frame, parameters)
        else:
            raise BacktestValidationError(f"Unsupported strategy type: {strategy_type}")

        performance = self._performance(frame, signals, initial_capital)
        return BacktestResult(performance=performance, signals=signals)

    def _moving_average_signals(self, frame: pd.DataFrame, parameters: dict[str, Any]) -> list[dict[str, Any]]:
        short_window = int(parameters.get("shortWindow", 5))
        long_window = int(parameters.get("longWindow", 20))
        if short_window <= 0 or long_window <= 0 or short_window >= long_window:
            raise BacktestValidationError("Moving average crossover requires shortWindow > 0 and shortWindow < longWindow.")
        if len(frame) < long_window + 1:
            raise BacktestValidationError(f"Not enough market data: need at least {long_window + 1} prices for moving average crossover.")

        data = frame.copy()
        data["short_ma"] = data["price"].rolling(short_window).mean()
        data["long_ma"] = data["price"].rolling(long_window).mean()
        data["above"] = data["short_ma"] > data["long_ma"]
        data["previous_above"] = data["above"].shift(1)

        signals: list[dict[str, Any]] = []
        for row in data.dropna().itertuples():
            if bool(row.above) and not bool(row.previous_above):
                signals.append(self._signal("BUY", row.price, row.event_time, "short moving average crossed above long moving average"))
            elif not bool(row.above) and bool(row.previous_above):
                signals.append(self._signal("SELL", row.price, row.event_time, "short moving average crossed below long moving average"))
        return signals or [self._final_hold(frame, "no moving average crossover detected")]

    def _rsi_signals(self, frame: pd.DataFrame, parameters: dict[str, Any]) -> list[dict[str, Any]]:
        period = int(parameters.get("period", 14))
        if period <= 1:
            raise BacktestValidationError("RSI requires period > 1.")
        if len(frame) < period + 1:
            raise BacktestValidationError(f"Not enough market data: need at least {period + 1} prices for RSI.")

        delta = frame["price"].diff()
        gain = delta.clip(lower=0).rolling(period).mean()
        loss = (-delta.clip(upper=0)).rolling(period).mean()
        rs = gain / loss.replace(0, np.nan)
        rsi = (100 - (100 / (1 + rs))).fillna(100)

        signals: list[dict[str, Any]] = []
        for idx, value in rsi.dropna().items():
            row = frame.loc[idx]
            if value < 30:
                signals.append(self._signal("BUY", row["price"], row["event_time"], "RSI below 30"))
            elif value > 70:
                signals.append(self._signal("SELL", row["price"], row["event_time"], "RSI above 70"))
            else:
                signals.append(self._signal("HOLD", row["price"], row["event_time"], "RSI neutral"))
        return signals[-10:] or [self._final_hold(frame, "RSI neutral")]

    def _volatility_breakout_signals(self, frame: pd.DataFrame, parameters: dict[str, Any]) -> list[dict[str, Any]]:
        window = int(parameters.get("window", 20))
        if window <= 1:
            raise BacktestValidationError("Volatility breakout requires window > 1.")
        if len(frame) < window + 1:
            raise BacktestValidationError(f"Not enough market data: need at least {window + 1} prices for volatility breakout.")

        data = frame.copy()
        data["rolling_high"] = data["price"].rolling(window).max().shift(1)
        data["rolling_low"] = data["price"].rolling(window).min().shift(1)

        signals: list[dict[str, Any]] = []
        for row in data.dropna().itertuples():
            if row.price > row.rolling_high:
                signals.append(self._signal("BUY", row.price, row.event_time, "price broke above rolling high"))
            elif row.price < row.rolling_low:
                signals.append(self._signal("SELL", row.price, row.event_time, "price broke below rolling low"))
        return signals or [self._final_hold(frame, "no volatility breakout detected")]

    def _performance(self, frame: pd.DataFrame, signals: list[dict[str, Any]], initial_capital: float) -> dict[str, Any]:
        prices = frame["price"].astype(float)
        returns = prices.pct_change().fillna(0)
        equity = initial_capital * (1 + returns).cumprod()
        total_return = ((equity.iloc[-1] - initial_capital) / initial_capital) * 100
        peak = equity.cummax()
        drawdowns = ((equity - peak) / peak) * 100
        max_drawdown = abs(float(drawdowns.min()))
        std = float(returns.std())
        sharpe = float((returns.mean() / std) * np.sqrt(252)) if std > 0 else 0.0
        trade_signals = [signal for signal in signals if signal["signal"] in {"BUY", "SELL"}]
        buy_prices = [signal["price"] for signal in trade_signals if signal["signal"] == "BUY"]
        sell_prices = [signal["price"] for signal in trade_signals if signal["signal"] == "SELL"]
        pairs = zip(buy_prices, sell_prices)
        outcomes = [sell > buy for buy, sell in pairs]
        win_rate = (sum(outcomes) / len(outcomes) * 100) if outcomes else 0.0

        return {
            "total_return": round(float(total_return), 4),
            "win_rate": round(float(win_rate), 4),
            "max_drawdown": round(max_drawdown, 4),
            "sharpe_ratio": round(sharpe, 4),
            "total_trades": len(trade_signals),
        }

    def _signal(self, signal: str, price: float, event_time: datetime, reason: str) -> dict[str, Any]:
        return {
            "signal": signal,
            "price": float(price),
            "event_time": event_time.to_pydatetime() if hasattr(event_time, "to_pydatetime") else event_time,
            "reason": reason,
        }

    def _final_hold(self, frame: pd.DataFrame, reason: str) -> dict[str, Any]:
        row = frame.iloc[-1]
        return self._signal("HOLD", row["price"], row["event_time"], reason)
