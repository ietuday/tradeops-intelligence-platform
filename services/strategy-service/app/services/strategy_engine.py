import time

from app.models import Strategy
from app.observability import metrics
from app.repositories.strategy_repository import StrategyRepository
from app.schemas import BacktestRequest
from app.services.backtest_engine import BacktestEngine, BacktestResult
from app.services.signal_service import SignalService


class StrategyEngine:
    def __init__(self, repo: StrategyRepository, backtest_engine: BacktestEngine, signal_service: SignalService):
        self.repo = repo
        self.backtest_engine = backtest_engine
        self.signal_service = signal_service

    def run_backtest(self, strategy: Strategy, request: BacktestRequest, correlation_id: str | None):
        metrics.backtests_started_total.inc()
        start = time.perf_counter()
        try:
            prices = self.repo.get_market_prices(strategy.symbol, request.startTime, request.endTime)
            result: BacktestResult = self.backtest_engine.run(
                strategy.strategy_type,
                strategy.parameters,
                prices,
                request.initialCapital,
            )
            run, signals, performance = self.repo.save_backtest(
                strategy,
                request.startTime,
                request.endTime,
                request.initialCapital,
                result.performance,
                result.signals,
                correlation_id,
            )
            self.signal_service.publish_backtest_events(strategy, run, signals)
            metrics.backtests_completed_total.inc()
            return run, signals, performance
        except Exception:
            metrics.backtests_failed_total.inc()
            raise
        finally:
            metrics.backtest_duration_seconds.observe(time.perf_counter() - start)
