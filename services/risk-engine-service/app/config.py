import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    service_port: int
    database_url: str
    kafka_brokers: str
    jwt_secret: str
    score_updated_topic: str
    breached_topic: str
    anomaly_topic: str
    recommendation_topic: str


def get_settings() -> Settings:
    return Settings(
        service_port=int(os.getenv("RISK_SERVICE_PORT", "8080")),
        database_url=os.getenv(
            "RISK_DATABASE_URL",
            "postgresql+psycopg://tradeops:tradeops@localhost:5432/tradeops",
        ),
        kafka_brokers=os.getenv("RISK_KAFKA_BROKERS", "redpanda:29092"),
        jwt_secret=os.getenv("RISK_JWT_SECRET", ""),
        score_updated_topic=os.getenv("RISK_SCORE_UPDATED_TOPIC", "risk.score.updated"),
        breached_topic=os.getenv("RISK_BREACHED_TOPIC", "risk.breached"),
        anomaly_topic=os.getenv("RISK_ANOMALY_TOPIC", "risk.anomaly.detected"),
        recommendation_topic=os.getenv("RISK_RECOMMENDATION_TOPIC", "risk.recommendation.created"),
    )
