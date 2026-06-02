from app.services.var_service import VarService


def test_var_returns_fallback_for_sparse_data():
    result = VarService().calculate([])

    assert result["value_at_risk"] == 1000
    assert result["method"] == "fallback"


def test_var_calculates_parametric_result():
    result = VarService().calculate([
        {"total_value": 100000},
        {"total_value": 99000},
        {"total_value": 101000},
        {"total_value": 98000},
    ])

    assert result["value_at_risk"] > 0
    assert result["method"] == "parametric"
