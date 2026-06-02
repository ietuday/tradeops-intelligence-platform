from fastapi import APIRouter, Depends, Response
from sqlalchemy import text
from sqlalchemy.orm import Session

from app.db import get_db
from app.kafka.producer import KafkaProducer
from app.main_dependencies import get_kafka_producer
from app.observability.metrics import metrics_response


router = APIRouter()


@router.get("/health")
def health():
    return {"service": "risk-engine-service", "status": "ok"}


@router.get("/ready")
def ready(db: Session = Depends(get_db), producer: KafkaProducer = Depends(get_kafka_producer)):
    db.execute(text("SELECT 1"))
    producer.ready()
    return {"service": "risk-engine-service", "status": "ready"}


@router.get("/metrics")
def metrics():
    body, content_type = metrics_response()
    return Response(content=body, media_type=content_type)
