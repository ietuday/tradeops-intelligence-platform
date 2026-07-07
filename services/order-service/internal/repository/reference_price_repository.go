package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReferencePrice struct {
	TenantID  string
	Symbol    string
	Price     float64
	Source    string
	UpdatedAt time.Time
}

type ReferencePriceRepository struct {
	db *pgxpool.Pool
}

func NewReferencePriceRepository(db *pgxpool.Pool) *ReferencePriceRepository {
	return &ReferencePriceRepository{db: db}
}

func (r *ReferencePriceRepository) UpsertReferencePrice(ctx context.Context, price ReferencePrice) error {
	price.TenantID = strings.TrimSpace(price.TenantID)
	price.Symbol = strings.ToUpper(strings.TrimSpace(price.Symbol))
	price.Source = strings.TrimSpace(price.Source)
	if price.TenantID == "" {
		return errors.New("tenantID is required")
	}
	if price.Symbol == "" {
		return errors.New("symbol is required")
	}
	if price.Price <= 0 {
		return errors.New("price must be positive")
	}
	if price.Source == "" {
		return errors.New("source is required")
	}
	if price.UpdatedAt.IsZero() {
		price.UpdatedAt = time.Now().UTC()
	} else {
		price.UpdatedAt = price.UpdatedAt.UTC()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO reference_prices (tenant_id, symbol, price, source, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, symbol) DO UPDATE
		SET price = EXCLUDED.price,
		    source = EXCLUDED.source,
		    updated_at = EXCLUDED.updated_at
	`, price.TenantID, price.Symbol, price.Price, price.Source, price.UpdatedAt)
	return err
}

func (r *ReferencePriceRepository) UpsertMarketReferencePrice(ctx context.Context, tenantID, symbol string, price float64, source string, updatedAt time.Time) error {
	return r.UpsertReferencePrice(ctx, ReferencePrice{
		TenantID:  tenantID,
		Symbol:    symbol,
		Price:     price,
		Source:    source,
		UpdatedAt: updatedAt,
	})
}

func (r *ReferencePriceRepository) GetReferencePrice(ctx context.Context, tenantID string, symbol string) (*ReferencePrice, error) {
	tenantID = strings.TrimSpace(tenantID)
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if tenantID == "" || symbol == "" {
		return nil, ErrNotFound
	}
	var price ReferencePrice
	err := r.db.QueryRow(ctx, `
		SELECT tenant_id, symbol, price::float8, source, updated_at
		FROM reference_prices
		WHERE tenant_id = $1 AND symbol = $2
	`, tenantID, symbol).Scan(&price.TenantID, &price.Symbol, &price.Price, &price.Source, &price.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &price, nil
}

func (r *ReferencePriceRepository) GetReferencePriceAge(ctx context.Context, tenantID string, symbol string) (time.Duration, error) {
	price, err := r.GetReferencePrice(ctx, tenantID, symbol)
	if err != nil {
		return 0, err
	}
	return time.Since(price.UpdatedAt.UTC()), nil
}
