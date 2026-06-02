from functools import lru_cache

from app.config import get_settings
from app.kafka.producer import KafkaProducer
from app.services.backtest_engine import BacktestEngine


@lru_cache(maxsize=1)
def get_kafka_producer() -> KafkaProducer:
    return KafkaProducer(get_settings())


@lru_cache(maxsize=1)
def get_backtest_engine() -> BacktestEngine:
    return BacktestEngine()
