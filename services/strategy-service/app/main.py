import os
import uuid

from fastapi import FastAPI, Request
from starlette.middleware.base import RequestResponseEndpoint
from starlette.responses import Response

from app.api.health import router as health_router
from app.api.routes import router as strategy_router
from app.db import run_migrations


def create_app(run_db_migrations: bool | None = None) -> FastAPI:
    app = FastAPI(title="TradeOps Strategy Service", version="0.6.0")

    @app.middleware("http")
    async def correlation_id_middleware(request: Request, call_next: RequestResponseEndpoint) -> Response:
        correlation_id = request.headers.get("x-correlation-id") or str(uuid.uuid4())
        request.state.correlation_id = correlation_id
        response = await call_next(request)
        response.headers["X-Correlation-ID"] = correlation_id
        return response

    app.include_router(health_router)
    app.include_router(strategy_router)

    @app.on_event("startup")
    def startup() -> None:
        should_run = run_db_migrations
        if should_run is None:
            should_run = os.getenv("STRATEGY_SKIP_MIGRATIONS", "").lower() not in {"1", "true", "yes"}
        if should_run:
            run_migrations()

    return app


app = create_app()
