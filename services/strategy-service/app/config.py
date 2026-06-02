import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    service_port: int
    database_url: str
    kafka_brokers: str
    signal_topic: str
    backtest_completed_topic: str
    jwt_secret: str


def get_settings() -> Settings:
    return Settings(
        service_port=int(os.getenv("STRATEGY_SERVICE_PORT", "8080")),
        database_url=os.getenv(
            "STRATEGY_DATABASE_URL",
            "postgresql+psycopg://tradeops:tradeops@localhost:5432/tradeops",
        ),
        kafka_brokers=os.getenv("STRATEGY_KAFKA_BROKERS", "redpanda:29092"),
        signal_topic=os.getenv("STRATEGY_SIGNAL_TOPIC", "strategy.signal.generated"),
        backtest_completed_topic=os.getenv("STRATEGY_BACKTEST_COMPLETED_TOPIC", "strategy.backtest.completed"),
        jwt_secret=os.getenv("STRATEGY_JWT_SECRET", ""),
    )
