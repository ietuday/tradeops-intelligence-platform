from datetime import datetime
from typing import Literal

from pydantic import BaseModel


RiskLevel = Literal["LOW", "MEDIUM", "HIGH", "CRITICAL"]


class RiskScoreResponse(BaseModel):
    score: float
    level: RiskLevel
    factors: dict[str, float]
    calculatedAt: datetime


class VolatilityItem(BaseModel):
    symbol: str
    volatility: float
    samples: int


class VolatilityResponse(BaseModel):
    symbols: list[VolatilityItem]
    calculatedAt: datetime


class DrawdownResponse(BaseModel):
    maxDrawdown: float
    currentDrawdown: float
    samples: int
    calculatedAt: datetime


class VarResponse(BaseModel):
    valueAtRisk: float
    confidenceLevel: float
    timeHorizonDays: int
    method: str
    calculatedAt: datetime


class RecommendationResponse(BaseModel):
    id: str
    type: str
    message: str
    severity: RiskLevel
    context: dict
    createdAt: datetime


class AnomalyResponse(BaseModel):
    id: str
    symbol: str
    type: str
    severity: RiskLevel
    value: float
    zScore: float
    eventTime: datetime
    createdAt: datetime
