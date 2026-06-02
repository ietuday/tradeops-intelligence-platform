import os

from fastapi import FastAPI

from app.api.health import router as health_router
from app.api.routes import router as risk_router
from app.db import run_migrations


def create_app(run_db_migrations: bool | None = None) -> FastAPI:
    app = FastAPI(title="TradeOps Risk Engine Service", version="0.7.0")
    app.include_router(health_router)
    app.include_router(risk_router)

    @app.on_event("startup")
    def startup() -> None:
        should_run = run_db_migrations
        if should_run is None:
            should_run = os.getenv("RISK_SKIP_MIGRATIONS", "").lower() not in {"1", "true", "yes"}
        if should_run:
            run_migrations()

    return app


app = create_app()
