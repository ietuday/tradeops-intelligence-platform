CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS market_ticks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  symbol TEXT NOT NULL,
  price NUMERIC NOT NULL,
  volume NUMERIC NOT NULL,
  source TEXT NOT NULL,
  event_time TIMESTAMPTZ NOT NULL,
  received_at TIMESTAMPTZ NOT NULL,
  correlation_id TEXT
);

CREATE TABLE IF NOT EXISTS ohlc_candles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  symbol TEXT NOT NULL,
  interval TEXT NOT NULL,
  open NUMERIC NOT NULL,
  high NUMERIC NOT NULL,
  low NUMERIC NOT NULL,
  close NUMERIC NOT NULL,
  volume NUMERIC NOT NULL,
  candle_start TIMESTAMPTZ NOT NULL,
  candle_end TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_market_ticks_symbol_received_at ON market_ticks(symbol, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_market_ticks_event_time ON market_ticks(event_time DESC);
CREATE INDEX IF NOT EXISTS idx_ohlc_candles_symbol_interval_start ON ohlc_candles(symbol, interval, candle_start DESC);
