import pytest
from fastapi import HTTPException
from fastapi.security import HTTPAuthorizationCredentials

from app.auth import current_user, require_write


def test_strategy_routes_require_bearer_token():
    with pytest.raises(HTTPException) as exc:
        current_user(None)
    assert exc.value.status_code == 401


def test_create_strategy_rejects_viewer(monkeypatch):
    monkeypatch.setenv("STRATEGY_JWT_SECRET", "secret")
    token = __import__("jwt").encode(
        {"sub": "user-1", "roles": ["viewer"], "iss": "tradeops-identity-service"},
        "secret",
        algorithm="HS256",
    )
    credentials = HTTPAuthorizationCredentials(scheme="Bearer", credentials=token)

    with pytest.raises(HTTPException) as exc:
        require_write(current_user(credentials))
    assert exc.value.status_code == 403
