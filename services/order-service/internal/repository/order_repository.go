package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrIdempotencyConflict = errors.New("idempotency conflict")

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindIdempotency(ctx context.Context, userID, key string) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	err := r.db.QueryRow(ctx, `
		SELECT user_id, key, request_hash, order_id::text, created_at
		FROM idempotency_keys
		WHERE user_id = $1 AND key = $2
	`, userID, key).Scan(&record.UserID, &record.Key, &record.RequestHash, &record.OrderID, &record.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.IdempotencyRecord{}, ErrNotFound
		}
		return domain.IdempotencyRecord{}, err
	}
	return record, nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order domain.Order, events []domain.OrderEvent, idempotencyKey, requestHash string) (domain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, symbol, side, order_type, quantity, limit_price, stop_price, status, fill_price, reject_reason, correlation_id, cancelled_at, filled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id::text, created_at, updated_at
	`, order.UserID, order.Symbol, order.Side, order.OrderType, order.Quantity, order.LimitPrice, order.StopPrice, order.Status, order.FillPrice, order.RejectReason, order.CorrelationID, order.CancelledAt, order.FilledAt).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return domain.Order{}, err
	}

	for i := range events {
		events[i].OrderID = order.ID
		if err := insertEvent(ctx, tx, events[i]); err != nil {
			return domain.Order{}, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO idempotency_keys (user_id, key, request_hash, order_id)
		VALUES ($1, $2, $3, $4)
	`, order.UserID, idempotencyKey, requestHash, order.ID); err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	var order domain.Order
	err := r.db.QueryRow(ctx, `
		SELECT id::text, user_id, symbol, side, order_type, quantity::float8, limit_price::float8, stop_price::float8, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
		FROM orders
		WHERE id = $1
	`, id).Scan(&order.ID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.LimitPrice, &order.StopPrice, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Order{}, ErrNotFound
		}
		return domain.Order{}, err
	}
	return order, nil
}

func (r *OrderRepository) ListOrders(ctx context.Context, userID string, includeAll bool) ([]domain.Order, error) {
	query := `
		SELECT id::text, user_id, symbol, side, order_type, quantity::float8, limit_price::float8, stop_price::float8, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
		FROM orders
	`
	args := []any{}
	if !includeAll {
		query += ` WHERE user_id = $1`
		args = append(args, userID)
	}
	query += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.LimitPrice, &order.StopPrice, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (r *OrderRepository) CancelOrder(ctx context.Context, id, correlationID string, event domain.OrderEvent) (domain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback(ctx)

	var order domain.Order
	err = tx.QueryRow(ctx, `
		UPDATE orders
		SET status = $2, cancelled_at = now(), updated_at = now(), correlation_id = $3
		WHERE id = $1 AND status = $4
		RETURNING id::text, user_id, symbol, side, order_type, quantity::float8, limit_price::float8, stop_price::float8, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
	`, id, domain.StatusCancelled, correlationID, domain.StatusAccepted).Scan(&order.ID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.LimitPrice, &order.StopPrice, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Order{}, ErrNotFound
		}
		return domain.Order{}, err
	}
	event.OrderID = order.ID
	event.UserID = order.UserID
	if err := insertEvent(ctx, tx, event); err != nil {
		return domain.Order{}, err
	}
	return order, tx.Commit(ctx)
}

type txQuerier interface {
	Exec(context.Context, string, ...any) (pgconnCommandTag, error)
}

type pgconnCommandTag interface{}

func insertEvent(ctx context.Context, tx pgx.Tx, event domain.OrderEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO order_events (event_id, event_type, order_id, user_id, payload, correlation_id, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.EventID, event.EventType, event.OrderID, event.UserID, payload, event.CorrelationID, event.OccurredAt)
	return err
}
