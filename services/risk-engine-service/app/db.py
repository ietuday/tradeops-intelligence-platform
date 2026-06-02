from pathlib import Path

from sqlalchemy import create_engine, text
from sqlalchemy.engine import Engine
from sqlalchemy.orm import sessionmaker

from app.config import get_settings


settings = get_settings()
engine = create_engine(settings.database_url, pool_pre_ping=True)
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)


def run_migrations(db_engine: Engine = engine) -> None:
    migration_path = Path(__file__).resolve().parents[1] / "migrations" / "001_create_risk_tables.sql"
    sql = migration_path.read_text(encoding="utf-8")
    with db_engine.begin() as conn:
        for statement in [part.strip() for part in sql.split(";") if part.strip()]:
            conn.execute(text(statement))


def get_db():
    session = SessionLocal()
    try:
        yield session
    finally:
        session.close()
