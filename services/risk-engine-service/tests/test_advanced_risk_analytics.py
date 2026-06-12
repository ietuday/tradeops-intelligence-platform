from app.schemas import (
    ConcentrationRiskRequest,
    DrawdownTrendRequest,
    HistoricalValue,
    Position,
    Scenario,
    StressTestRequest,
    VolatilityShockRequest,
)
from app.services.advanced_risk_analytics_service import AdvancedRiskAnalyticsService


def test_stress_test_market_drop():
    result = AdvancedRiskAnalyticsService().stress_test(
        StressTestRequest(
            portfolioId="demo",
            positions=[position("AAPL", 10, 180)],
            scenarios=[Scenario(name="Market drops 10%", marketShockPercent=-10)],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.baselineValue == 1800
    assert result.stressedValue == 1620
    assert result.pnlImpactPercent == -10
    assert result.worstScenario == "Market drops 10%"


def test_symbol_shock_overrides_sector_and_market_shock():
    result = AdvancedRiskAnalyticsService().stress_test(
        StressTestRequest(
            portfolioId="demo",
            positions=[position("AAPL", 10, 100, sector="Technology")],
            scenarios=[
                Scenario(
                    name="Override",
                    marketShockPercent=-5,
                    sectorShocks={"Technology": -15},
                    symbolShocks={"AAPL": -25},
                )
            ],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.scenarioResults[0].pnlImpactPercent == -25


def test_sector_shock_overrides_market_shock():
    result = AdvancedRiskAnalyticsService().stress_test(
        StressTestRequest(
            portfolioId="demo",
            positions=[position("MSFT", 10, 100, sector="Technology")],
            scenarios=[Scenario(name="Sector", marketShockPercent=-5, sectorShocks={"Technology": -15})],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.scenarioResults[0].pnlImpactPercent == -15


def test_liquidity_haircut_applied():
    result = AdvancedRiskAnalyticsService().stress_test(
        StressTestRequest(
            portfolioId="demo",
            positions=[position("AAPL", 10, 100)],
            scenarios=[Scenario(name="Haircut", marketShockPercent=-10, liquidityHaircutPercent=10)],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.stressedValue == 810
    assert result.pnlImpactPercent == -19


def test_worst_scenario_selected_correctly():
    result = AdvancedRiskAnalyticsService().stress_test(
        StressTestRequest(
            portfolioId="demo",
            positions=[position("AAPL", 10, 100)],
            scenarios=[
                Scenario(name="Small", marketShockPercent=-5),
                Scenario(name="Large", marketShockPercent=-20),
            ],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.worstScenario == "Large"
    assert result.stressedValue == 800


def test_concentration_risk_score_and_level():
    result = AdvancedRiskAnalyticsService().concentration(
        ConcentrationRiskRequest(
            portfolioId="demo",
            positions=[
                position("AAPL", 7, 100, sector="Technology"),
                position("JPM", 3, 100, sector="Financials"),
            ],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.concentrationScore == 70
    assert result.riskLevel == "CRITICAL"
    assert result.recommendations[0].code == "REDUCE_CONCENTRATION"


def test_drawdown_max_calculation():
    result = AdvancedRiskAnalyticsService().drawdown_trend(
        DrawdownTrendRequest(
            portfolioId="demo",
            values=[
                HistoricalValue(value=100),
                HistoricalValue(value=120),
                HistoricalValue(value=90),
                HistoricalValue(value=110),
            ],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.peakValue == 120
    assert result.troughValue == 90
    assert result.maxDrawdownPercent == 25
    assert result.riskLevel == "CRITICAL"


def test_volatility_shock_calculation_and_recommendation():
    result = AdvancedRiskAnalyticsService().volatility_shock(
        VolatilityShockRequest(
            portfolioId="demo",
            positions=[position("AAPL", 10, 100)],
            volatilityMultiplier=2,
            baseRiskScore=30,
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.stressedValue == 900
    assert result.shockedRiskScore == 60
    assert result.recommendations[0].code == "REDUCE_VOLATILITY_EXPOSURE"


def test_zero_baseline_handled_safely():
    result = AdvancedRiskAnalyticsService().stress_test(
        StressTestRequest(
            portfolioId="empty",
            positions=[],
            scenarios=[Scenario(name="Market drops 10%", marketShockPercent=-10)],
        ),
        "tenant-a",
        "corr-1",
    )

    assert result.baselineValue == 0
    assert result.pnlImpactPercent == 0


def test_tenant_and_correlation_returned():
    result = AdvancedRiskAnalyticsService().concentration(
        ConcentrationRiskRequest(portfolioId="demo", positions=[position("AAPL", 1, 100)]),
        "tenant-a",
        "corr-1",
    )

    assert result.tenantId == "tenant-a"
    assert result.correlationId == "corr-1"


def test_invalid_negative_quantity_rejected():
    try:
        position("AAPL", -1, 100)
    except ValueError as error:
        assert "quantity" in str(error)
    else:
        raise AssertionError("negative quantity should fail validation")


def position(symbol: str, quantity: float, price: float, sector: str = "Technology") -> Position:
    return Position(
        symbol=symbol,
        quantity=quantity,
        averagePrice=price,
        currentPrice=price,
        sector=sector,
        assetClass="EQUITY",
    )
