from app.services.drawdown_service import DrawdownService


def test_drawdown_calculates_max_and_current_drawdown():
    result = DrawdownService().calculate([
        {"total_value": 100},
        {"total_value": 120},
        {"total_value": 90},
        {"total_value": 110},
    ])

    assert round(result["max_drawdown"], 2) == 25.0
    assert round(result["current_drawdown"], 2) == 8.33
