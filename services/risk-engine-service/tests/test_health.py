from app.api.health import health


def test_health_is_public():
    assert health() == {"service": "risk-engine-service", "status": "ok"}
