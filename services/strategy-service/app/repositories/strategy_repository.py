from datetime import datetime
from typing import Any

from sqlalchemy import text
from sqlalchemy.orm import Session

from app import models


class StrategyRepository:
    def __init__(self, db: Session):
        self.db = db

    def create_strategy(self, user_id: str, name: str, symbol: str, strategy_type: str, parameters: dict[str, Any]) -> models.Strategy:
        strategy = models.Strategy(
            user_id=user_id,
            name=name,
            symbol=symbol,
            strategy_type=strategy_type,
            parameters=parameters,
        )
        self.db.add(strategy)
        self.db.commit()
        self.db.refresh(strategy)
        return strategy

    def list_strategies(self, user_id: str) -> list[models.Strategy]:
        return self.db.query(models.Strategy).filter(models.Strategy.user_id == user_id).order_by(models.Strategy.created_at.desc()).all()

    def get_strategy(self, user_id: str, strategy_id: str) -> models.Strategy | None:
        return (
            self.db.query(models.Strategy)
            .filter(models.Strategy.id == strategy_id, models.Strategy.user_id == user_id)
            .one_or_none()
        )

    def get_market_prices(self, symbol: str, start_time: datetime, end_time: datetime) -> list[dict[str, Any]]:
        rows = self.db.execute(
            text(
                """
                SELECT price::float8 AS price, event_time
                FROM market_ticks
                WHERE symbol = :symbol
                  AND event_time >= :start_time
                  AND event_time <= :end_time
                ORDER BY event_time ASC
                """
            ),
            {"symbol": symbol, "start_time": start_time, "end_time": end_time},
        )
        return [{"price": row.price, "event_time": row.event_time} for row in rows]

    def save_backtest(
        self,
        strategy: models.Strategy,
        start_time: datetime,
        end_time: datetime,
        initial_capital: float,
        performance: dict[str, Any],
        signals: list[dict[str, Any]],
        correlation_id: str | None,
    ) -> tuple[models.BacktestRun, list[models.StrategySignal], models.StrategyPerformance]:
        run = models.BacktestRun(
            strategy_id=strategy.id,
            user_id=strategy.user_id,
            start_time=start_time,
            end_time=end_time,
            initial_capital=initial_capital,
            total_return=performance["total_return"],
            win_rate=performance["win_rate"],
            max_drawdown=performance["max_drawdown"],
            sharpe_ratio=performance["sharpe_ratio"],
            total_trades=performance["total_trades"],
        )
        self.db.add(run)
        self.db.flush()

        saved_signals = [
            models.StrategySignal(
                strategy_id=strategy.id,
                backtest_run_id=run.id,
                user_id=strategy.user_id,
                symbol=strategy.symbol,
                signal=signal["signal"],
                price=signal["price"],
                reason=signal["reason"],
                event_time=signal["event_time"],
                correlation_id=correlation_id,
            )
            for signal in signals
        ]
        self.db.add_all(saved_signals)

        existing = (
            self.db.query(models.StrategyPerformance)
            .filter(models.StrategyPerformance.strategy_id == strategy.id)
            .one_or_none()
        )
        if existing:
            existing.total_return = performance["total_return"]
            existing.win_rate = performance["win_rate"]
            existing.max_drawdown = performance["max_drawdown"]
            existing.sharpe_ratio = performance["sharpe_ratio"]
            existing.total_trades = performance["total_trades"]
            existing.updated_at = models.now_utc()
            perf = existing
        else:
            perf = models.StrategyPerformance(
                strategy_id=strategy.id,
                user_id=strategy.user_id,
                total_return=performance["total_return"],
                win_rate=performance["win_rate"],
                max_drawdown=performance["max_drawdown"],
                sharpe_ratio=performance["sharpe_ratio"],
                total_trades=performance["total_trades"],
            )
            self.db.add(perf)

        self.db.commit()
        self.db.refresh(run)
        for signal in saved_signals:
            self.db.refresh(signal)
        self.db.refresh(perf)
        return run, saved_signals, perf

    def get_performance(self, user_id: str, strategy_id: str) -> models.StrategyPerformance | None:
        return (
            self.db.query(models.StrategyPerformance)
            .filter(models.StrategyPerformance.strategy_id == strategy_id, models.StrategyPerformance.user_id == user_id)
            .one_or_none()
        )

    def list_signals(self, user_id: str, strategy_id: str) -> list[models.StrategySignal]:
        return (
            self.db.query(models.StrategySignal)
            .filter(models.StrategySignal.strategy_id == strategy_id, models.StrategySignal.user_id == user_id)
            .order_by(models.StrategySignal.event_time.desc())
            .limit(100)
            .all()
        )
