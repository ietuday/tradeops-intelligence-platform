from fastapi import APIRouter, Depends, Header, HTTPException, status
from sqlalchemy.orm import Session

from app.auth import UserContext, require_read, require_write
from app.db import get_db
from app.main_dependencies import get_backtest_engine, get_kafka_producer
from app.observability import metrics
from app.repositories.strategy_repository import StrategyRepository
from app.schemas import BacktestRequest, BacktestResponse, PerformanceResponse, SignalResponse, StrategyCreate, StrategyResponse
from app.services.backtest_engine import BacktestEngine, BacktestValidationError
from app.services.signal_service import SignalService
from app.services.strategy_engine import StrategyEngine


router = APIRouter()


def repo(db: Session = Depends(get_db)) -> StrategyRepository:
    return StrategyRepository(db)


@router.post("/strategies", response_model=StrategyResponse, status_code=status.HTTP_201_CREATED, response_model_by_alias=False)
def create_strategy(payload: StrategyCreate, user: UserContext = Depends(require_write), repository: StrategyRepository = Depends(repo)):
    strategy = repository.create_strategy(
        user.user_id,
        payload.name,
        payload.symbol,
        payload.strategyType.value,
        payload.parameters,
    )
    metrics.strategies_created_total.inc()
    return strategy


@router.get("/strategies", response_model=list[StrategyResponse], response_model_by_alias=False)
def list_strategies(user: UserContext = Depends(require_read), repository: StrategyRepository = Depends(repo)):
    return repository.list_strategies(user.user_id)


@router.get("/strategies/{strategy_id}", response_model=StrategyResponse, response_model_by_alias=False)
def get_strategy(strategy_id: str, user: UserContext = Depends(require_read), repository: StrategyRepository = Depends(repo)):
    strategy = repository.get_strategy(user.user_id, strategy_id)
    if not strategy:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Strategy not found.")
    return strategy


@router.post("/strategies/{strategy_id}/backtest", response_model=BacktestResponse)
def run_backtest(
    strategy_id: str,
    payload: BacktestRequest,
    user: UserContext = Depends(require_write),
    repository: StrategyRepository = Depends(repo),
    backtest_engine: BacktestEngine = Depends(get_backtest_engine),
    kafka_producer=Depends(get_kafka_producer),
    x_correlation_id: str | None = Header(default=None),
):
    strategy = repository.get_strategy(user.user_id, strategy_id)
    if not strategy:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Strategy not found.")
    engine = StrategyEngine(repository, backtest_engine, SignalService(kafka_producer))
    try:
        run, signals, performance = engine.run_backtest(strategy, payload, x_correlation_id)
    except BacktestValidationError as exc:
        raise HTTPException(status_code=status.HTTP_422_UNPROCESSABLE_ENTITY, detail=str(exc)) from exc
    return BacktestResponse(
        id=run.id,
        strategyId=strategy.id,
        performance=to_performance(performance),
        signals=[to_signal(signal) for signal in signals],
        createdAt=run.created_at,
    )


@router.get("/strategies/{strategy_id}/performance", response_model=PerformanceResponse)
def get_performance(strategy_id: str, user: UserContext = Depends(require_read), repository: StrategyRepository = Depends(repo)):
    if not repository.get_strategy(user.user_id, strategy_id):
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Strategy not found.")
    performance = repository.get_performance(user.user_id, strategy_id)
    if not performance:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Strategy performance not found.")
    return to_performance(performance)


@router.get("/strategies/{strategy_id}/signals", response_model=list[SignalResponse])
def get_signals(strategy_id: str, user: UserContext = Depends(require_read), repository: StrategyRepository = Depends(repo)):
    if not repository.get_strategy(user.user_id, strategy_id):
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Strategy not found.")
    return [to_signal(signal) for signal in repository.list_signals(user.user_id, strategy_id)]


def to_performance(performance) -> PerformanceResponse:
    return PerformanceResponse(
        totalReturn=performance.total_return,
        winRate=performance.win_rate,
        maxDrawdown=performance.max_drawdown,
        sharpeRatio=performance.sharpe_ratio,
        totalTrades=performance.total_trades,
    )


def to_signal(signal) -> SignalResponse:
    return SignalResponse(
        id=signal.id,
        strategyId=signal.strategy_id,
        backtestRunId=signal.backtest_run_id,
        symbol=signal.symbol,
        signal=signal.signal,
        price=signal.price,
        reason=signal.reason,
        eventTime=signal.event_time,
    )
