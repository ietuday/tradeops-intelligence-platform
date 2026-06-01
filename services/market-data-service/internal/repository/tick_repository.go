package repository

import (
	"context"

	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TickRepository struct {
	db *pgxpool.Pool
}

func NewTickRepository(db *pgxpool.Pool) *TickRepository {
	return &TickRepository{db: db}
}

func (r *TickRepository) StoreTick(ctx context.Context, tick domain.Tick) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO market_ticks (symbol, price, volume, source, event_time, received_at, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tick.Symbol, tick.Price, tick.Volume, tick.Source, tick.EventTime, tick.ReceivedAt, tick.CorrelationID)
	return err
}

func (r *TickRepository) LatestTicks(ctx context.Context, limit int) ([]domain.Tick, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (symbol) symbol, price::float8, volume::float8, source, event_time, received_at, COALESCE(correlation_id, '')
		FROM market_ticks
		ORDER BY symbol, received_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ticks []domain.Tick
	for rows.Next() {
		var tick domain.Tick
		if err := rows.Scan(&tick.Symbol, &tick.Price, &tick.Volume, &tick.Source, &tick.EventTime, &tick.ReceivedAt, &tick.CorrelationID); err != nil {
			return nil, err
		}
		ticks = append(ticks, tick)
	}
	return ticks, rows.Err()
}

func (r *TickRepository) Symbols(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT symbol FROM market_ticks ORDER BY symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	return symbols, rows.Err()
}
