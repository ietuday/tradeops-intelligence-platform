from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.auth import UserContext, require_full_read, require_read
from app.db import get_db
from app.kafka.producer import KafkaProducer
from app.main_dependencies import (
    get_anomaly_service,
    get_drawdown_service,
    get_kafka_producer,
    get_recommendation_service,
    get_risk_score_service,
    get_var_service,
    get_volatility_service,
)
from app.observability import metrics
from app.repositories.risk_repository import RiskRepository
from app.schemas import AnomalyResponse, DrawdownResponse, RecommendationResponse, RiskScoreResponse, VarResponse, VolatilityResponse
from app.services.anomaly_service import AnomalyService
from app.services.drawdown_service import DrawdownService
from app.services.recommendation_service import RecommendationService
from app.services.risk_score_service import RiskScoreService
from app.services.var_service import VarService
from app.services.volatility_service import VolatilityService


router = APIRouter()


def repo(db: Session = Depends(get_db)) -> RiskRepository:
    return RiskRepository(db)


@router.get("/risk/portfolio/score", response_model=RiskScoreResponse)
def risk_score(
    user: UserContext = Depends(require_read),
    repository: RiskRepository = Depends(repo),
    volatility_service: VolatilityService = Depends(get_volatility_service),
    drawdown_service: DrawdownService = Depends(get_drawdown_service),
    score_service: RiskScoreService = Depends(get_risk_score_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    score = calculate_score(repository, user.user_id, volatility_service, drawdown_service, score_service)
    saved = repository.save_score(user.user_id, score["score"], score["level"], score["factors"])
    metrics.risk_scores_calculated_total.inc()
    metrics.risk_score_current.set(score["score"])
    producer.publish_score_updated({"eventType": "risk.score.updated", "userId": user.user_id, "score": score["score"], "level": score["level"], "riskScoreId": saved.id})
    if score["level"] in {"HIGH", "CRITICAL"}:
        metrics.risk_breaches_total.inc()
        producer.publish_breached({"eventType": "risk.breached", "userId": user.user_id, "score": score["score"], "level": score["level"]})
    return to_score_response(score)


@router.get("/risk/portfolio/volatility", response_model=VolatilityResponse)
def volatility(
    user: UserContext = Depends(require_full_read),
    repository: RiskRepository = Depends(repo),
    volatility_service: VolatilityService = Depends(get_volatility_service),
):
    holdings = repository.get_holdings(user.user_id)
    ticks = repository.get_market_ticks([item["symbol"] for item in holdings])
    result = volatility_service.calculate(ticks)
    return VolatilityResponse(symbols=result["symbols"], calculatedAt=result["calculated_at"])


@router.get("/risk/portfolio/drawdown", response_model=DrawdownResponse)
def drawdown(
    user: UserContext = Depends(require_full_read),
    repository: RiskRepository = Depends(repo),
    drawdown_service: DrawdownService = Depends(get_drawdown_service),
):
    result = drawdown_service.calculate(repository.get_snapshots(user.user_id))
    metrics.drawdown_current.set(result["max_drawdown"])
    return DrawdownResponse(maxDrawdown=result["max_drawdown"], currentDrawdown=result["current_drawdown"], samples=result["samples"], calculatedAt=result["calculated_at"])


@router.get("/risk/portfolio/var", response_model=VarResponse)
def value_at_risk(
    user: UserContext = Depends(require_full_read),
    repository: RiskRepository = Depends(repo),
    var_service: VarService = Depends(get_var_service),
):
    result = var_service.calculate(repository.get_snapshots(user.user_id))
    metrics.var_current.set(result["value_at_risk"])
    return to_var_response(result)


@router.get("/risk/recommendations", response_model=list[RecommendationResponse])
def recommendations(
    user: UserContext = Depends(require_read),
    repository: RiskRepository = Depends(repo),
    volatility_service: VolatilityService = Depends(get_volatility_service),
    drawdown_service: DrawdownService = Depends(get_drawdown_service),
    var_service: VarService = Depends(get_var_service),
    score_service: RiskScoreService = Depends(get_risk_score_service),
    anomaly_service: AnomalyService = Depends(get_anomaly_service),
    recommendation_service: RecommendationService = Depends(get_recommendation_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    holdings = repository.get_holdings(user.user_id)
    ticks = repository.get_market_ticks([item["symbol"] for item in holdings])
    volatility_result = volatility_service.calculate(ticks)
    drawdown_result = drawdown_service.calculate(repository.get_snapshots(user.user_id))
    var_result = var_service.calculate(repository.get_snapshots(user.user_id))
    score = calculate_score(repository, user.user_id, volatility_service, drawdown_service, score_service)
    anomalies = anomaly_service.detect(ticks)
    generated = recommendation_service.generate(score, volatility_result, drawdown_result, var_result, anomalies)
    saved = repository.save_recommendations(user.user_id, generated)
    for item in saved:
        metrics.risk_recommendations_created_total.inc()
        producer.publish_recommendation({"eventType": "risk.recommendation.created", "userId": user.user_id, "recommendationId": item.id, "type": item.recommendation_type})
    return [to_recommendation_response(item) for item in saved]


@router.get("/risk/anomalies", response_model=list[AnomalyResponse])
def anomalies(
    user: UserContext = Depends(require_full_read),
    repository: RiskRepository = Depends(repo),
    anomaly_service: AnomalyService = Depends(get_anomaly_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    holdings = repository.get_holdings(user.user_id)
    detected = anomaly_service.detect(repository.get_market_ticks([item["symbol"] for item in holdings]))
    saved = repository.save_anomalies(user.user_id, detected) if detected else repository.latest_anomalies(user.user_id)
    for item in saved:
        metrics.risk_anomalies_detected_total.inc()
        producer.publish_anomaly({"eventType": "risk.anomaly.detected", "userId": user.user_id, "anomalyId": item.id, "symbol": item.symbol})
    return [to_anomaly_response(item) for item in saved]


def calculate_score(repository: RiskRepository, user_id: str, volatility_service: VolatilityService, drawdown_service: DrawdownService, score_service: RiskScoreService):
    portfolio = repository.get_portfolio(user_id)
    holdings = repository.get_holdings(user_id)
    symbols = [item["symbol"] for item in holdings]
    volatility_result = volatility_service.calculate(repository.get_market_ticks(symbols))
    drawdown_result = drawdown_service.calculate(repository.get_snapshots(user_id))
    latest_prices = repository.get_latest_prices(symbols)
    return score_service.calculate(portfolio, holdings, latest_prices, volatility_result, drawdown_result, repository.count_recent_anomalies(user_id))


def to_score_response(score: dict) -> RiskScoreResponse:
    return RiskScoreResponse(score=score["score"], level=score["level"], factors=score["factors"], calculatedAt=score["calculated_at"])


def to_var_response(result: dict) -> VarResponse:
    return VarResponse(valueAtRisk=result["value_at_risk"], confidenceLevel=result["confidence_level"], timeHorizonDays=result["time_horizon_days"], method=result["method"], calculatedAt=result["calculated_at"])


def to_recommendation_response(item) -> RecommendationResponse:
    return RecommendationResponse(id=item.id, type=item.recommendation_type, message=item.message, severity=item.severity, context=item.context, createdAt=item.created_at)


def to_anomaly_response(item) -> AnomalyResponse:
    return AnomalyResponse(id=item.id, symbol=item.symbol, type=item.anomaly_type, severity=item.severity, value=item.value, zScore=item.z_score, eventTime=item.event_time, createdAt=item.created_at)
