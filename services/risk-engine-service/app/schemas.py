from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field, field_validator


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


class Position(BaseModel):
    symbol: str
    quantity: float
    averagePrice: float
    currentPrice: float
    assetClass: str | None = None
    sector: str | None = None

    @field_validator("quantity")
    @classmethod
    def quantity_must_be_non_negative(cls, value: float) -> float:
        if value < 0:
            raise ValueError("quantity must be non-negative")
        return value

    @field_validator("averagePrice", "currentPrice")
    @classmethod
    def prices_must_be_non_negative(cls, value: float) -> float:
        if value < 0:
            raise ValueError("price must be non-negative")
        return value


class Scenario(BaseModel):
    name: str
    marketShockPercent: float | None = None
    volatilityMultiplier: float | None = None
    symbolShocks: dict[str, float] = Field(default_factory=dict)
    sectorShocks: dict[str, float] = Field(default_factory=dict)
    liquidityHaircutPercent: float | None = None


class RiskRecommendation(BaseModel):
    code: str
    severity: RiskLevel
    message: str
    suggestedAction: str


class ScenarioResult(BaseModel):
    scenarioName: str
    baselineValue: float
    stressedValue: float
    pnlImpact: float
    pnlImpactPercent: float
    riskLevel: RiskLevel
    affectedSymbols: list[str]


class StressTestRequest(BaseModel):
    portfolioId: str
    positions: list[Position] = Field(default_factory=list)
    scenarios: list[Scenario]
    tenantId: str | None = None
    correlationId: str | None = None


class StressTestResult(BaseModel):
    portfolioId: str
    tenantId: str
    correlationId: str
    baselineValue: float
    stressedValue: float
    pnlImpact: float
    pnlImpactPercent: float
    worstScenario: str | None
    scenarioResults: list[ScenarioResult]
    recommendations: list[RiskRecommendation]
    generatedAt: datetime


class ScenarioRunRequest(BaseModel):
    portfolioId: str
    positions: list[Position] = Field(default_factory=list)
    scenarioNames: list[str]
    tenantId: str | None = None
    correlationId: str | None = None


class ConcentrationRiskRequest(BaseModel):
    portfolioId: str
    positions: list[Position] = Field(default_factory=list)
    tenantId: str | None = None
    correlationId: str | None = None


class ExposureItem(BaseModel):
    name: str
    value: float
    exposurePercent: float


class ConcentrationRiskResult(BaseModel):
    portfolioId: str
    tenantId: str
    correlationId: str
    totalValue: float
    topPositions: list[ExposureItem]
    sectorExposure: dict[str, float]
    assetClassExposure: dict[str, float]
    concentrationScore: float
    riskLevel: RiskLevel
    recommendations: list[RiskRecommendation]
    generatedAt: datetime


class HistoricalValue(BaseModel):
    timestamp: datetime | None = None
    value: float

    @field_validator("value")
    @classmethod
    def value_must_be_non_negative(cls, value: float) -> float:
        if value < 0:
            raise ValueError("historical value must be non-negative")
        return value


class DrawdownTrendRequest(BaseModel):
    portfolioId: str
    values: list[HistoricalValue] = Field(default_factory=list)
    tenantId: str | None = None
    correlationId: str | None = None


class DrawdownObservation(BaseModel):
    value: float
    peakValue: float
    drawdownPercent: float


class DrawdownTrendResult(BaseModel):
    portfolioId: str
    tenantId: str
    correlationId: str
    peakValue: float
    troughValue: float
    maxDrawdown: float
    maxDrawdownPercent: float
    observations: list[DrawdownObservation]
    trend: str
    riskLevel: RiskLevel
    recommendations: list[RiskRecommendation]
    generatedAt: datetime


class VolatilityShockRequest(BaseModel):
    portfolioId: str
    positions: list[Position] = Field(default_factory=list)
    volatilityMultiplier: float
    baseRiskScore: float = 25.0
    tenantId: str | None = None
    correlationId: str | None = None

    @field_validator("volatilityMultiplier")
    @classmethod
    def multiplier_must_be_positive(cls, value: float) -> float:
        if value <= 0:
            raise ValueError("volatilityMultiplier must be positive")
        return value


class VolatilityShockResult(BaseModel):
    portfolioId: str
    tenantId: str
    correlationId: str
    baselineValue: float
    stressedValue: float
    pnlImpact: float
    pnlImpactPercent: float
    volatilityMultiplier: float
    shockedRiskScore: float
    riskLevel: RiskLevel
    recommendations: list[RiskRecommendation]
    generatedAt: datetime
