from dataclasses import dataclass

import jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from app.config import get_settings


ALLOWED_READ_ROLES = {"trading_admin", "trader", "analyst", "risk_manager", "viewer"}
ALLOWED_WRITE_ROLES = {"trading_admin", "trader", "analyst", "risk_manager"}

bearer_scheme = HTTPBearer(auto_error=False)


@dataclass(frozen=True)
class UserContext:
    user_id: str
    roles: list[str]


def current_user(credentials: HTTPAuthorizationCredentials | None = Depends(bearer_scheme)) -> UserContext:
    if credentials is None or credentials.scheme.lower() != "bearer":
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Bearer token is required.")
    settings = get_settings()
    if not settings.jwt_secret:
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail="STRATEGY_JWT_SECRET is not configured.")
    try:
        payload = jwt.decode(
            credentials.credentials,
            settings.jwt_secret,
            algorithms=["HS256"],
            issuer="tradeops-identity-service",
        )
    except jwt.PyJWTError:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid bearer token.") from None
    user_id = payload.get("sub")
    roles = payload.get("roles") or []
    if not user_id or not isinstance(roles, list):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid bearer token claims.")
    return UserContext(user_id=str(user_id), roles=[str(role) for role in roles])


def require_read(user: UserContext = Depends(current_user)) -> UserContext:
    if not set(user.roles).intersection(ALLOWED_READ_ROLES):
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Role is not allowed to read strategies.")
    return user


def require_write(user: UserContext = Depends(current_user)) -> UserContext:
    if not set(user.roles).intersection(ALLOWED_WRITE_ROLES):
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Role is not allowed to change strategies.")
    return user
