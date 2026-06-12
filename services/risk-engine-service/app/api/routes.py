from time import perf_counter

from fastapi import APIRouter, Depends, HTTPException, Request
from sqlalchemy.orm import Session

from app.auth import UserContext, require_full_read, require_read
from app.db import get_db
from app.kafka.producer import KafkaProducer
from app.main_dependencies import (
    get_advanced_risk_analytics_service,
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
from app.schemas import (
    AnomalyResponse,
    ConcentrationRiskRequest,
    ConcentrationRiskResult,
    DrawdownResponse,
    DrawdownTrendRequest,
    DrawdownTrendResult,
    HistoricalValue,
    RecommendationResponse,
    RiskScoreResponse,
    ScenarioRunRequest,
    StressTestRequest,
    StressTestResult,
    VarResponse,
    VolatilityResponse,
    VolatilityShockRequest,
    VolatilityShockResult,
)
from app.services.advanced_risk_analytics_service import AdvancedRiskAnalyticsService
from app.services.anomaly_service import AnomalyService
from app.services.drawdown_service import DrawdownService
from app.services.recommendation_service import RecommendationService
from app.services.risk_score_service import RiskScoreService
from app.services.var_service import VarService
from app.services.volatility_service import VolatilityService


router = APIRouter()
DEFAULT_TENANT_ID = "default-tenant"


def repo(db: Session = Depends(get_db)) -> RiskRepository:
    return RiskRepository(db)


@router.get("/risk/portfolio/score", response_model=RiskScoreResponse)
def risk_score(
    request: Request,
    user: UserContext = Depends(require_read),
    repository: RiskRepository = Depends(repo),
    volatility_service: VolatilityService = Depends(get_volatility_service),
    drawdown_service: DrawdownService = Depends(get_drawdown_service),
    score_service: RiskScoreService = Depends(get_risk_score_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    correlation_id = getattr(request.state, "correlation_id", None)
    score = calculate_score(repository, user.user_id, volatility_service, drawdown_service, score_service)
    saved = repository.save_score(user.user_id, score["score"], score["level"], score["factors"])
    metrics.risk_scores_calculated_total.inc()
    metrics.risk_score_current.set(score["score"])
    producer.publish_score_updated({"eventType": "risk.score.updated", "userId": user.user_id, "score": score["score"], "level": score["level"], "riskScoreId": saved.id, "correlationId": correlation_id})
    if score["level"] in {"HIGH", "CRITICAL"}:
        metrics.risk_breaches_total.inc()
        producer.publish_breached({"eventType": "risk.breached", "userId": user.user_id, "score": score["score"], "level": score["level"], "correlationId": correlation_id})
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
    request: Request,
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
    correlation_id = getattr(request.state, "correlation_id", None)
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
        producer.publish_recommendation({"eventType": "risk.recommendation.created", "userId": user.user_id, "recommendationId": item.id, "type": item.recommendation_type, "correlationId": correlation_id})
    return [to_recommendation_response(item) for item in saved]


@router.get("/risk/anomalies", response_model=list[AnomalyResponse])
def anomalies(
    request: Request,
    user: UserContext = Depends(require_full_read),
    repository: RiskRepository = Depends(repo),
    anomaly_service: AnomalyService = Depends(get_anomaly_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    correlation_id = getattr(request.state, "correlation_id", None)
    holdings = repository.get_holdings(user.user_id)
    detected = anomaly_service.detect(repository.get_market_ticks([item["symbol"] for item in holdings]))
    saved = repository.save_anomalies(user.user_id, detected) if detected else repository.latest_anomalies(user.user_id)
    for item in saved:
        metrics.risk_anomalies_detected_total.inc()
        producer.publish_anomaly({"eventType": "risk.anomaly.detected", "userId": user.user_id, "anomalyId": item.id, "symbol": item.symbol, "correlationId": correlation_id})
    return [to_anomaly_response(item) for item in saved]


@router.get("/api/v1/risk/scenarios")
def built_in_scenarios(
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
):
    return {"scenarios": analytics.built_in_scenarios()}


@router.post("/api/v1/risk/stress-test", response_model=StressTestResult)
def stress_test(
    payload: StressTestRequest,
    request: Request,
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    del user
    tenant_id, correlation_id = tenant_and_correlation(request, payload.tenantId, payload.correlationId)
    with metrics.risk_analytics_duration_seconds.labels(operation="stress_test").time():
        result = analytics.stress_test(payload, tenant_id, correlation_id)
    metrics.risk_stress_tests_total.labels(status="success").inc()
    for recommendation in result.recommendations:
        metrics.risk_recommendations_generated_total.labels(severity=recommendation.severity).inc()
    producer.publish_stress_test_completed(risk_analytics_event("risk.stress_test.completed", result))
    return result


@router.post("/api/v1/risk/scenarios/run", response_model=StressTestResult)
def run_scenarios(
    payload: ScenarioRunRequest,
    request: Request,
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    del user
    scenarios = analytics.scenarios_by_name(payload.scenarioNames)
    if len(scenarios) != len(payload.scenarioNames):
        known = {scenario.name for scenario in scenarios}
        unknown = [name for name in payload.scenarioNames if name not in known]
        raise HTTPException(status_code=400, detail={"message": "Unknown scenario name.", "unknownScenarios": unknown})
    tenant_id, correlation_id = tenant_and_correlation(request, payload.tenantId, payload.correlationId)
    stress_payload = StressTestRequest(portfolioId=payload.portfolioId, positions=payload.positions, scenarios=scenarios, tenantId=tenant_id, correlationId=correlation_id)
    with metrics.risk_analytics_duration_seconds.labels(operation="scenario_run").time():
        result = analytics.stress_test(stress_payload, tenant_id, correlation_id)
    for scenario in result.scenarioResults:
        metrics.risk_scenarios_run_total.labels(scenario=scenario.scenarioName, status="success").inc()
    for recommendation in result.recommendations:
        metrics.risk_recommendations_generated_total.labels(severity=recommendation.severity).inc()
    producer.publish_scenario_completed(risk_analytics_event("risk.scenario.completed", result))
    return result


@router.get("/api/v1/risk/portfolio/{portfolio_id}/concentration", response_model=ConcentrationRiskResult)
def concentration_placeholder(
    portfolio_id: str,
    request: Request,
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    del user
    tenant_id, correlation_id = tenant_and_correlation(request)
    payload = ConcentrationRiskRequest(portfolioId=portfolio_id, positions=demo_positions(), tenantId=tenant_id, correlationId=correlation_id)
    return concentration(payload, request, analytics=analytics, producer=producer)


@router.post("/api/v1/risk/portfolio/concentration", response_model=ConcentrationRiskResult)
def concentration(
    payload: ConcentrationRiskRequest,
    request: Request,
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    del user
    tenant_id, correlation_id = tenant_and_correlation(request, payload.tenantId, payload.correlationId)
    with metrics.risk_analytics_duration_seconds.labels(operation="concentration").time():
        result = analytics.concentration(payload, tenant_id, correlation_id)
    metrics.risk_concentration_analyses_total.labels(status="success").inc()
    for recommendation in result.recommendations:
        metrics.risk_recommendations_generated_total.labels(severity=recommendation.severity).inc()
    producer.publish_concentration_analyzed(risk_analytics_event("risk.concentration.analyzed", result))
    return result


@router.get("/api/v1/risk/portfolio/{portfolio_id}/drawdown-trend", response_model=DrawdownTrendResult)
def drawdown_trend_placeholder(
    portfolio_id: str,
    request: Request,
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    del user
    tenant_id, correlation_id = tenant_and_correlation(request)
    payload = DrawdownTrendRequest(portfolioId=portfolio_id, values=demo_history(), tenantId=tenant_id, correlationId=correlation_id)
    return drawdown_trend(payload, request, analytics=analytics, producer=producer)


@router.post("/api/v1/risk/portfolio/drawdown-trend", response_model=DrawdownTrendResult)
def drawdown_trend(
    payload: DrawdownTrendRequest,
    request: Request,
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
    producer: KafkaProducer = Depends(get_kafka_producer),
):
    del user
    tenant_id, correlation_id = tenant_and_correlation(request, payload.tenantId, payload.correlationId)
    with metrics.risk_analytics_duration_seconds.labels(operation="drawdown_trend").time():
        result = analytics.drawdown_trend(payload, tenant_id, correlation_id)
    metrics.risk_drawdown_analyses_total.labels(status="success").inc()
    for recommendation in result.recommendations:
        metrics.risk_recommendations_generated_total.labels(severity=recommendation.severity).inc()
    producer.publish_drawdown_analyzed(risk_analytics_event("risk.drawdown.analyzed", result))
    return result


@router.post("/api/v1/risk/volatility-shock", response_model=VolatilityShockResult)
def volatility_shock(
    payload: VolatilityShockRequest,
    request: Request,
    user: UserContext = Depends(require_read),
    analytics: AdvancedRiskAnalyticsService = Depends(get_advanced_risk_analytics_service),
):
    del user
    tenant_id, correlation_id = tenant_and_correlation(request, payload.tenantId, payload.correlationId)
    with metrics.risk_analytics_duration_seconds.labels(operation="volatility_shock").time():
        result = analytics.volatility_shock(payload, tenant_id, correlation_id)
    for recommendation in result.recommendations:
        metrics.risk_recommendations_generated_total.labels(severity=recommendation.severity).inc()
    return result


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


def tenant_and_correlation(request: Request, tenant_id: str | None = None, correlation_id: str | None = None) -> tuple[str, str]:
    resolved_tenant = tenant_id or request.headers.get("x-tenant-id") or DEFAULT_TENANT_ID
    resolved_correlation = correlation_id or getattr(request.state, "correlation_id", None) or request.headers.get("x-correlation-id") or ""
    return resolved_tenant, resolved_correlation


def risk_analytics_event(event_type: str, result) -> dict:
    payload = {
        "eventType": event_type,
        "eventVersion": "1.0",
        "tenantId": result.tenantId,
        "correlationId": result.correlationId,
        "portfolioId": result.portfolioId,
        "riskLevel": result.riskLevel if hasattr(result, "riskLevel") else worst_risk_level(result),
        "worstScenario": getattr(result, "worstScenario", None),
        "pnlImpactPercent": getattr(result, "pnlImpactPercent", 0.0),
        "generatedAt": result.generatedAt,
    }
    if hasattr(result, "concentrationScore"):
        payload["concentrationScore"] = result.concentrationScore
    if hasattr(result, "maxDrawdownPercent"):
        payload["maxDrawdownPercent"] = result.maxDrawdownPercent
    return payload


def worst_risk_level(result: StressTestResult) -> str:
    order = {"LOW": 0, "MEDIUM": 1, "HIGH": 2, "CRITICAL": 3}
    worst = max(result.scenarioResults, key=lambda item: order[item.riskLevel], default=None)
    return worst.riskLevel if worst else "LOW"


def demo_positions():
    return [
        {"symbol": "AAPL", "quantity": 10, "averagePrice": 150, "currentPrice": 180, "sector": "Technology", "assetClass": "EQUITY"},
        {"symbol": "MSFT", "quantity": 8, "averagePrice": 250, "currentPrice": 300, "sector": "Technology", "assetClass": "EQUITY"},
        {"symbol": "JPM", "quantity": 5, "averagePrice": 140, "currentPrice": 150, "sector": "Financials", "assetClass": "EQUITY"},
    ]


def demo_history():
    return [
        HistoricalValue(value=10000),
        HistoricalValue(value=11200),
        HistoricalValue(value=9800),
        HistoricalValue(value=9100),
        HistoricalValue(value=10400),
    ]
