from prometheus_client import CONTENT_TYPE_LATEST, Counter, Histogram, generate_latest


strategies_created_total = Counter("strategies_created_total", "Total strategies created.")
backtests_started_total = Counter("backtests_started_total", "Total backtests started.")
backtests_completed_total = Counter("backtests_completed_total", "Total backtests completed.")
backtests_failed_total = Counter("backtests_failed_total", "Total backtests failed.")
strategy_signals_generated_total = Counter("strategy_signals_generated_total", "Total strategy signals generated.")
backtest_duration_seconds = Histogram("backtest_duration_seconds", "Backtest duration in seconds.")
kafka_publish_errors_total = Counter("kafka_publish_errors_total", "Total Kafka publish errors.")


def metrics_response() -> tuple[bytes, str]:
    return generate_latest(), CONTENT_TYPE_LATEST
