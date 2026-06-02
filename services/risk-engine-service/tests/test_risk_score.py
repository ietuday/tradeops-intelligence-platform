from app.services.risk_score_service import RiskScoreService, risk_level


def test_risk_score_uses_expected_level_bands():
    assert risk_level(30) == "LOW"
    assert risk_level(31) == "MEDIUM"
    assert risk_level(61) == "HIGH"
    assert risk_level(81) == "CRITICAL"


def test_risk_score_increases_with_concentration_and_low_cash():
    result = RiskScoreService().calculate(
        {"cash_balance": 1000},
        [{"symbol": "AAPL", "quantity": 10, "average_buy_price": 100}],
        {"AAPL": 100},
        {"symbols": [{"symbol": "AAPL", "volatility": 40, "samples": 10}]},
        {"max_drawdown": 12},
        2,
    )

    assert result["score"] > 20
    assert result["level"] in {"LOW", "MEDIUM", "HIGH", "CRITICAL"}
    assert "portfolioConcentration" in result["factors"]
