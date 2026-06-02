from functools import lru_cache

from app.config import get_settings
from app.kafka.producer import KafkaProducer
from app.services.anomaly_service import AnomalyService
from app.services.drawdown_service import DrawdownService
from app.services.recommendation_service import RecommendationService
from app.services.risk_score_service import RiskScoreService
from app.services.var_service import VarService
from app.services.volatility_service import VolatilityService


@lru_cache(maxsize=1)
def get_kafka_producer() -> KafkaProducer:
    return KafkaProducer(get_settings())


def get_volatility_service() -> VolatilityService:
    return VolatilityService()


def get_drawdown_service() -> DrawdownService:
    return DrawdownService()


def get_var_service() -> VarService:
    return VarService()


def get_anomaly_service() -> AnomalyService:
    return AnomalyService()


def get_risk_score_service() -> RiskScoreService:
    return RiskScoreService()


def get_recommendation_service() -> RecommendationService:
    return RecommendationService()
