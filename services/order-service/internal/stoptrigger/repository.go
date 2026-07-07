package stoptrigger

import (
	"context"
	"time"
)

type Repository interface {
	TriggerDueStopOrders(ctx context.Context, limit int, maxReferencePriceAge time.Duration, now time.Time) (BatchResult, error)
}

type ReferencePriceStore interface {
	UpsertMarketReferencePrice(ctx context.Context, tenantID, symbol string, price float64, source string, updatedAt time.Time) error
}
