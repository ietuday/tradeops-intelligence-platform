from datetime import datetime
from enum import Enum
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator


class StrategyType(str, Enum):
    moving_average_crossover = "MOVING_AVERAGE_CROSSOVER"
    rsi = "RSI"
    volatility_breakout = "VOLATILITY_BREAKOUT"


class StrategyCreate(BaseModel):
    name: str = Field(min_length=1, max_length=120)
    symbol: str = Field(min_length=1, max_length=32)
    strategyType: StrategyType
    parameters: dict[str, Any] = Field(default_factory=dict)

    @field_validator("symbol")
    @classmethod
    def normalize_symbol(cls, value: str) -> str:
        return value.strip().upper()


class StrategyResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    name: str
    symbol: str
    strategyType: StrategyType = Field(alias="strategy_type")
    parameters: dict[str, Any]
    createdAt: datetime = Field(alias="created_at")
    updatedAt: datetime = Field(alias="updated_at")


class BacktestRequest(BaseModel):
    startTime: datetime
    endTime: datetime
    initialCapital: float = Field(gt=0)

    @field_validator("endTime")
    @classmethod
    def validate_window(cls, value: datetime, values):
        start = values.data.get("startTime")
        if start and value <= start:
            raise ValueError("endTime must be after startTime")
        return value


class PerformanceResponse(BaseModel):
    totalReturn: float
    winRate: float
    maxDrawdown: float
    sharpeRatio: float
    totalTrades: int


class SignalResponse(BaseModel):
    id: str
    strategyId: str
    backtestRunId: str | None
    symbol: str
    signal: Literal["BUY", "SELL", "HOLD"]
    price: float
    reason: str
    eventTime: datetime


class BacktestResponse(BaseModel):
    id: str
    strategyId: str
    performance: PerformanceResponse
    signals: list[SignalResponse]
    createdAt: datetime
