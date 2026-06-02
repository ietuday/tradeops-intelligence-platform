CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS strategies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  name TEXT NOT NULL,
  symbol TEXT NOT NULL,
  strategy_type TEXT NOT NULL,
  parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS backtest_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ NOT NULL,
  initial_capital DOUBLE PRECISION NOT NULL,
  total_return DOUBLE PRECISION NOT NULL,
  win_rate DOUBLE PRECISION NOT NULL,
  max_drawdown DOUBLE PRECISION NOT NULL,
  sharpe_ratio DOUBLE PRECISION NOT NULL,
  total_trades INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS strategy_signals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
  backtest_run_id UUID REFERENCES backtest_runs(id) ON DELETE SET NULL,
  user_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  signal TEXT NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  reason TEXT NOT NULL,
  event_time TIMESTAMPTZ NOT NULL,
  correlation_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS strategy_performance (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  strategy_id UUID NOT NULL UNIQUE REFERENCES strategies(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  total_return DOUBLE PRECISION NOT NULL,
  win_rate DOUBLE PRECISION NOT NULL,
  max_drawdown DOUBLE PRECISION NOT NULL,
  sharpe_ratio DOUBLE PRECISION NOT NULL,
  total_trades INTEGER NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_strategies_user_id ON strategies(user_id);
CREATE INDEX IF NOT EXISTS idx_strategies_symbol ON strategies(symbol);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_strategy_id ON backtest_runs(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_signals_strategy_id ON strategy_signals(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_signals_symbol_time ON strategy_signals(symbol, event_time DESC);
