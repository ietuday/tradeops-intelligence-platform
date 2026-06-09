import json
import uuid
from typing import Any

from confluent_kafka import Producer
from confluent_kafka.admin import AdminClient, NewTopic

from app.config import Settings
from app.observability import metrics


class KafkaProducer:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.producer = Producer({"bootstrap.servers": settings.kafka_brokers})
        self.admin = AdminClient({"bootstrap.servers": settings.kafka_brokers})

    def ready(self) -> bool:
        self.ensure_topics()
        self.producer.list_topics(timeout=2)
        return True

    def ensure_topics(self) -> None:
        topics = [
            self.settings.score_updated_topic,
            self.settings.breached_topic,
            self.settings.anomaly_topic,
            self.settings.recommendation_topic,
        ]
        futures = self.admin.create_topics([NewTopic(topic, num_partitions=1, replication_factor=1) for topic in topics])
        for future in futures.values():
            try:
                future.result(timeout=2)
            except Exception:
                pass

    def publish_score_updated(self, payload: dict[str, Any]) -> None:
        self._publish(self.settings.score_updated_topic, payload)

    def publish_breached(self, payload: dict[str, Any]) -> None:
        self._publish(self.settings.breached_topic, payload)

    def publish_anomaly(self, payload: dict[str, Any]) -> None:
        self._publish(self.settings.anomaly_topic, payload)

    def publish_recommendation(self, payload: dict[str, Any]) -> None:
        self._publish(self.settings.recommendation_topic, payload)

    def _publish(self, topic: str, payload: dict[str, Any]) -> None:
        try:
            payload.setdefault("correlationId", str(uuid.uuid4()))
            self.producer.produce(topic, json.dumps(payload, default=str).encode("utf-8"))
            self.producer.poll(0)
            self.producer.flush(2)
        except Exception:
            metrics.kafka_publish_errors_total.inc()
