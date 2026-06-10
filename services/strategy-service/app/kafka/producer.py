import json
import uuid
from typing import Any

from confluent_kafka import Producer

from app.config import Settings
from app.observability import metrics


class KafkaProducer:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.producer = Producer({"bootstrap.servers": settings.kafka_brokers})

    def ready(self) -> bool:
        self.producer.list_topics(timeout=2)
        return True

    def publish_signal(self, payload: dict[str, Any]) -> None:
        self._publish(self.settings.signal_topic, payload)

    def publish_backtest_completed(self, payload: dict[str, Any]) -> None:
        self._publish(self.settings.backtest_completed_topic, payload)

    def _publish(self, topic: str, payload: dict[str, Any]) -> None:
        try:
            payload.setdefault("eventType", topic)
            payload.setdefault("eventVersion", "1.0")
            payload.setdefault("correlationId", str(uuid.uuid4()))
            self.producer.produce(topic, json.dumps(payload, default=str).encode("utf-8"))
            self.producer.poll(0)
            self.producer.flush(2)
        except Exception:
            metrics.kafka_publish_errors_total.inc()
