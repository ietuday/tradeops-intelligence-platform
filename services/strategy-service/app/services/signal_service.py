from app.kafka.producer import KafkaProducer
from app.models import BacktestRun, Strategy, StrategySignal
from app.observability import metrics


class SignalService:
    def __init__(self, producer: KafkaProducer):
        self.producer = producer

    def publish_backtest_events(self, strategy: Strategy, run: BacktestRun, signals: list[StrategySignal]) -> None:
        for signal in signals:
            self.producer.publish_signal(
                {
                    "eventType": "strategy.signal.generated",
                    "signalId": signal.id,
                    "strategyId": strategy.id,
                    "backtestRunId": run.id,
                    "userId": strategy.user_id,
                    "symbol": strategy.symbol,
                    "signal": signal.signal,
                    "price": signal.price,
                    "reason": signal.reason,
                    "eventTime": signal.event_time,
                    "correlationId": signal.correlation_id,
                }
            )
            metrics.strategy_signals_generated_total.inc()
        self.producer.publish_backtest_completed(
            {
                "eventType": "strategy.backtest.completed",
                "strategyId": strategy.id,
                "backtestRunId": run.id,
                "userId": strategy.user_id,
                "symbol": strategy.symbol,
                "createdAt": run.created_at,
            }
        )
