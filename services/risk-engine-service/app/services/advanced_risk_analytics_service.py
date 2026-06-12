from datetime import datetime, timezone
from typing import Iterable

from app.schemas import (
    ConcentrationRiskRequest,
    ConcentrationRiskResult,
    DrawdownObservation,
    DrawdownTrendRequest,
    DrawdownTrendResult,
    ExposureItem,
    Position,
    RiskRecommendation,
    RiskLevel,
    Scenario,
    ScenarioResult,
    StressTestRequest,
    StressTestResult,
    VolatilityShockRequest,
    VolatilityShockResult,
)


DEFAULT_TENANT_ID = "default-tenant"


BUILT_IN_SCENARIOS: dict[str, Scenario] = {
    "MARKET_DROP_5": Scenario(name="MARKET_DROP_5", marketShockPercent=-5),
    "MARKET_DROP_10": Scenario(name="MARKET_DROP_10", marketShockPercent=-10),
    "MARKET_DROP_20": Scenario(name="MARKET_DROP_20", marketShockPercent=-20),
    "TECH_SECTOR_DROP_15": Scenario(name="TECH_SECTOR_DROP_15", marketShockPercent=0, sectorShocks={"Technology": -15}),
    "VOLATILITY_DOUBLES": Scenario(name="VOLATILITY_DOUBLES", volatilityMultiplier=2),
    "LIQUIDITY_HAIRCUT_10": Scenario(name="LIQUIDITY_HAIRCUT_10", marketShockPercent=0, liquidityHaircutPercent=10),
    "SINGLE_SYMBOL_DROP_20": Scenario(name="SINGLE_SYMBOL_DROP_20", marketShockPercent=0, symbolShocks={"AAPL": -20}),
}


class AdvancedRiskAnalyticsService:
    def stress_test(self, request: StressTestRequest, tenant_id: str, correlation_id: str) -> StressTestResult:
        baseline = portfolio_value(request.positions)
        scenario_results = [self._run_scenario(scenario, request.positions, baseline) for scenario in request.scenarios]
        worst = min(scenario_results, key=lambda item: item.pnlImpactPercent, default=None)
        recommendations = recommendations_for_scenarios(scenario_results)
        return StressTestResult(
            portfolioId=request.portfolioId,
            tenantId=tenant_id,
            correlationId=correlation_id,
            baselineValue=round(baseline, 4),
            stressedValue=worst.stressedValue if worst else round(baseline, 4),
            pnlImpact=worst.pnlImpact if worst else 0.0,
            pnlImpactPercent=worst.pnlImpactPercent if worst else 0.0,
            worstScenario=worst.scenarioName if worst else None,
            scenarioResults=scenario_results,
            recommendations=recommendations,
            generatedAt=now_utc(),
        )

    def built_in_scenarios(self) -> list[Scenario]:
        return list(BUILT_IN_SCENARIOS.values())

    def scenarios_by_name(self, names: Iterable[str]) -> list[Scenario]:
        return [BUILT_IN_SCENARIOS[name] for name in names if name in BUILT_IN_SCENARIOS]

    def concentration(self, request: ConcentrationRiskRequest, tenant_id: str, correlation_id: str) -> ConcentrationRiskResult:
        total = portfolio_value(request.positions)
        by_symbol = exposure_by(request.positions, lambda position: position.symbol)
        by_sector = exposure_by(request.positions, lambda position: position.sector or "Unclassified")
        by_asset = exposure_by(request.positions, lambda position: position.assetClass or "UNKNOWN")
        top_symbol = max(percentages(by_symbol, total).values(), default=0.0)
        top_sector = max(percentages(by_sector, total).values(), default=0.0)
        score = round(max(top_symbol, top_sector), 4)
        level = concentration_level(score)
        return ConcentrationRiskResult(
            portfolioId=request.portfolioId,
            tenantId=tenant_id,
            correlationId=correlation_id,
            totalValue=round(total, 4),
            topPositions=[
                ExposureItem(name=name, value=round(value, 4), exposurePercent=round(exposure, 4))
                for name, value, exposure in sorted_exposures(by_symbol, total)
            ],
            sectorExposure={key: round(value, 4) for key, value in percentages(by_sector, total).items()},
            assetClassExposure={key: round(value, 4) for key, value in percentages(by_asset, total).items()},
            concentrationScore=score,
            riskLevel=level,
            recommendations=recommendations_for_concentration(score),
            generatedAt=now_utc(),
        )

    def drawdown_trend(self, request: DrawdownTrendRequest, tenant_id: str, correlation_id: str) -> DrawdownTrendResult:
        values = [item.value for item in request.values]
        if not values:
            return DrawdownTrendResult(
                portfolioId=request.portfolioId,
                tenantId=tenant_id,
                correlationId=correlation_id,
                peakValue=0.0,
                troughValue=0.0,
                maxDrawdown=0.0,
                maxDrawdownPercent=0.0,
                observations=[],
                trend="NO_DATA",
                riskLevel="LOW",
                recommendations=[],
                generatedAt=now_utc(),
            )

        peak = 0.0
        observations: list[DrawdownObservation] = []
        max_drawdown_percent = 0.0
        trough_value = values[0]
        for value in values:
            peak = max(peak, value)
            drawdown_percent = ((value - peak) / peak * 100) if peak else 0.0
            if drawdown_percent < max_drawdown_percent:
                max_drawdown_percent = drawdown_percent
                trough_value = value
            observations.append(DrawdownObservation(value=round(value, 4), peakValue=round(peak, 4), drawdownPercent=round(drawdown_percent, 4)))

        absolute_drawdown = abs(max_drawdown_percent)
        level = drawdown_level(absolute_drawdown)
        return DrawdownTrendResult(
            portfolioId=request.portfolioId,
            tenantId=tenant_id,
            correlationId=correlation_id,
            peakValue=round(max(values), 4),
            troughValue=round(trough_value, 4),
            maxDrawdown=round(absolute_drawdown, 4),
            maxDrawdownPercent=round(absolute_drawdown, 4),
            observations=observations,
            trend=trend_from_values(values),
            riskLevel=level,
            recommendations=recommendations_for_drawdown(absolute_drawdown),
            generatedAt=now_utc(),
        )

    def volatility_shock(self, request: VolatilityShockRequest, tenant_id: str, correlation_id: str) -> VolatilityShockResult:
        baseline = portfolio_value(request.positions)
        loss_factor = min(0.5, 0.05 * request.volatilityMultiplier)
        stressed = baseline * (1 - loss_factor)
        pnl = stressed - baseline
        pnl_percent = percent(pnl, baseline)
        shocked_score = min(100.0, request.baseRiskScore * request.volatilityMultiplier)
        level = score_level(shocked_score)
        return VolatilityShockResult(
            portfolioId=request.portfolioId,
            tenantId=tenant_id,
            correlationId=correlation_id,
            baselineValue=round(baseline, 4),
            stressedValue=round(stressed, 4),
            pnlImpact=round(pnl, 4),
            pnlImpactPercent=round(pnl_percent, 4),
            volatilityMultiplier=request.volatilityMultiplier,
            shockedRiskScore=round(shocked_score, 4),
            riskLevel=level,
            recommendations=recommendations_for_volatility(request.volatilityMultiplier),
            generatedAt=now_utc(),
        )

    def _run_scenario(self, scenario: Scenario, positions: list[Position], baseline: float) -> ScenarioResult:
        affected: list[str] = []
        stressed_value = 0.0
        for position in positions:
            shock = shock_for_position(scenario, position)
            if shock != 0:
                affected.append(position.symbol)
            stressed_value += position.quantity * position.currentPrice * (1 + shock / 100)
        if scenario.liquidityHaircutPercent:
            stressed_value *= 1 - scenario.liquidityHaircutPercent / 100
            affected = [position.symbol for position in positions] if positions else affected
        if scenario.volatilityMultiplier and scenario.volatilityMultiplier != 1:
            stressed_value *= 1 - min(0.5, 0.05 * scenario.volatilityMultiplier)
            affected = [position.symbol for position in positions] if positions else affected
        pnl = stressed_value - baseline
        pnl_percent = percent(pnl, baseline)
        return ScenarioResult(
            scenarioName=scenario.name,
            baselineValue=round(baseline, 4),
            stressedValue=round(stressed_value, 4),
            pnlImpact=round(pnl, 4),
            pnlImpactPercent=round(pnl_percent, 4),
            riskLevel=loss_level(abs(pnl_percent)),
            affectedSymbols=sorted(set(affected)),
        )


def portfolio_value(positions: list[Position]) -> float:
    return sum(position.quantity * position.currentPrice for position in positions)


def shock_for_position(scenario: Scenario, position: Position) -> float:
    if position.symbol in scenario.symbolShocks:
        return scenario.symbolShocks[position.symbol]
    if position.sector and position.sector in scenario.sectorShocks:
        return scenario.sectorShocks[position.sector]
    return scenario.marketShockPercent or 0.0


def exposure_by(positions: list[Position], key_fn) -> dict[str, float]:
    exposures: dict[str, float] = {}
    for position in positions:
        key = key_fn(position)
        exposures[key] = exposures.get(key, 0.0) + position.quantity * position.currentPrice
    return exposures


def percentages(values: dict[str, float], total: float) -> dict[str, float]:
    if total <= 0:
        return {key: 0.0 for key in values}
    return {key: value / total * 100 for key, value in values.items()}


def sorted_exposures(values: dict[str, float], total: float) -> list[tuple[str, float, float]]:
    pct = percentages(values, total)
    return sorted(((key, value, pct[key]) for key, value in values.items()), key=lambda item: item[2], reverse=True)


def percent(value: float, baseline: float) -> float:
    if baseline <= 0:
        return 0.0
    return value / baseline * 100


def loss_level(loss_percent: float) -> RiskLevel:
    if loss_percent > 20:
        return "CRITICAL"
    if loss_percent > 10:
        return "HIGH"
    if loss_percent > 5:
        return "MEDIUM"
    return "LOW"


def concentration_level(score: float) -> RiskLevel:
    if score > 60:
        return "CRITICAL"
    if score >= 40:
        return "HIGH"
    if score >= 25:
        return "MEDIUM"
    return "LOW"


def drawdown_level(drawdown_percent: float) -> RiskLevel:
    if drawdown_percent > 20:
        return "CRITICAL"
    if drawdown_percent > 10:
        return "HIGH"
    if drawdown_percent > 5:
        return "MEDIUM"
    return "LOW"


def score_level(score: float) -> RiskLevel:
    if score > 80:
        return "CRITICAL"
    if score > 60:
        return "HIGH"
    if score > 30:
        return "MEDIUM"
    return "LOW"


def recommendations_for_scenarios(results: list[ScenarioResult]) -> list[RiskRecommendation]:
    recommendations: list[RiskRecommendation] = []
    worst_loss = max((abs(result.pnlImpactPercent) for result in results if result.pnlImpactPercent < 0), default=0.0)
    if worst_loss > 10:
        recommendations.append(RiskRecommendation(
            code="MARKET_DOWNSIDE_SENSITIVITY",
            severity=loss_level(worst_loss),
            message="Portfolio is sensitive to downside shock scenarios.",
            suggestedAction="Review hedges, stop-losses, and position sizing before increasing exposure.",
        ))
    return recommendations


def recommendations_for_concentration(score: float) -> list[RiskRecommendation]:
    if score > 40:
        return [RiskRecommendation(
            code="REDUCE_CONCENTRATION",
            severity=concentration_level(score),
            message="Portfolio exposure is concentrated in a top symbol or sector.",
            suggestedAction="Reduce exposure to the top symbol or sector and diversify across uncorrelated holdings.",
        )]
    return []


def recommendations_for_drawdown(drawdown_percent: float) -> list[RiskRecommendation]:
    if drawdown_percent > 20:
        return [RiskRecommendation(
            code="REVIEW_DRAWDOWN_CONTROLS",
            severity=drawdown_level(drawdown_percent),
            message="Portfolio drawdown exceeds critical local threshold.",
            suggestedAction="Review stop-loss and hedging strategy before adding risk.",
        )]
    return []


def recommendations_for_volatility(multiplier: float) -> list[RiskRecommendation]:
    if multiplier >= 2:
        return [RiskRecommendation(
            code="REDUCE_VOLATILITY_EXPOSURE",
            severity="HIGH",
            message="Portfolio risk score is sensitive to volatility expansion.",
            suggestedAction="Consider reducing leveraged or high-volatility positions.",
        )]
    return []


def trend_from_values(values: list[float]) -> str:
    if len(values) < 2:
        return "FLAT"
    if values[-1] > values[0]:
        return "RECOVERING"
    if values[-1] < values[0]:
        return "DETERIORATING"
    return "FLAT"


def now_utc() -> datetime:
    return datetime.now(timezone.utc)
