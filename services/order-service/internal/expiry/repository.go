package expiry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
)

var ErrNoEligibleOrder = errors.New("order is no longer eligible for expiry")

type Repository struct {
	db       *pgxpool.Pool
	workerID string
}

func NewRepository(db *pgxpool.Pool, workerID string) *Repository {
	return &Repository{db: db, workerID: workerID}
}

func (r *Repository) ExpireDueBatch(ctx context.Context, limit int, now time.Time) (BatchResult, error) {
	tracer := otel.Tracer("order-service")
	ctx, span := tracer.Start(ctx, "order_expiry.process_batch")
	defer span.End()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return BatchResult{}, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(tenant_id, 'default-tenant'), time_in_force, expires_at
		FROM orders
		WHERE expires_at IS NOT NULL
		  AND expires_at <= $1
		  AND remaining_quantity > 0
		  AND status IN ($2, $3)
		  AND time_in_force IN ($4, $5)
		ORDER BY expires_at ASC, id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $6
	`, now, domain.StatusAccepted, domain.StatusPartiallyFilled, domain.TimeInForceDay, domain.TimeInForceGTD, limit)
	if err != nil {
		return BatchResult{}, err
	}
	defer rows.Close()

	var due []DueOrder
	for rows.Next() {
		var order DueOrder
		if err := rows.Scan(&order.ID, &order.TenantID, &order.TimeInForce, &order.ExpiresAt); err != nil {
			return BatchResult{}, err
		}
		due = append(due, order)
	}
	if err := rows.Err(); err != nil {
		return BatchResult{}, err
	}

	result := BatchResult{Processed: len(due), ExpiredByReason: map[string]map[string]int{}}
	for _, dueOrder := range due {
		order, err := r.expireLockedOrder(ctx, tx, dueOrder.ID, now)
		if errors.Is(err, ErrNoEligibleOrder) {
			result.Conflicts++
			continue
		}
		if err != nil {
			return BatchResult{}, err
		}
		if order != nil && order.ExpiryReason != nil {
			result.Expired++
			if result.ExpiredByReason[order.TimeInForce] == nil {
				result.ExpiredByReason[order.TimeInForce] = map[string]int{}
			}
			result.ExpiredByReason[order.TimeInForce][*order.ExpiryReason]++
			if order.ExpiresAt != nil && order.ExpiredAt != nil {
				result.LagSeconds = append(result.LagSeconds, order.ExpiredAt.Sub(*order.ExpiresAt).Seconds())
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return BatchResult{}, err
	}
	return result, nil
}

func (r *Repository) expireLockedOrder(ctx context.Context, tx pgx.Tx, id string, now time.Time) (*domain.Order, error) {
	tracer := otel.Tracer("order-service")
	ctx, span := tracer.Start(ctx, "order_expiry.expire_order")
	defer span.End()

	var order domain.Order
	err := tx.QueryRow(ctx, `
		UPDATE orders
		SET status = $2,
		    expired_at = $3,
		    expiry_reason = CASE time_in_force WHEN $4 THEN $5 ELSE $6 END,
		    expiry_worker_id = $7,
		    updated_at = $3,
		    correlation_id = $8,
		    version = version + 1
		WHERE id = $1
		  AND expires_at IS NOT NULL
		  AND expires_at <= $3
		  AND remaining_quantity > 0
		  AND status IN ($9, $10)
		  AND time_in_force IN ($4, $11)
		RETURNING id::text, COALESCE(tenant_id, 'default-tenant'), user_id, symbol, side, order_type,
		          quantity::float8, filled_quantity::float8, remaining_quantity::float8,
		          limit_price::float8, stop_price::float8, average_fill_price::float8,
		          time_in_force, expires_at, version, risk_decision_id::text, last_execution_at,
		          status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''),
		          created_at, updated_at, cancelled_at, filled_at, expired_at, expiry_reason
	`, id, domain.StatusExpired, now, domain.TimeInForceDay, ReasonDaySessionClosed, ReasonGTDReached, r.workerID, correlationID(id, now), domain.StatusAccepted, domain.StatusPartiallyFilled, domain.TimeInForceGTD).Scan(
		&order.ID, &order.TenantID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType,
		&order.Quantity, &order.FilledQuantity, &order.RemainingQuantity,
		&order.LimitPrice, &order.StopPrice, &order.AverageFillPrice,
		&order.TimeInForce, &order.ExpiresAt, &order.Version, &order.RiskDecisionID, &order.LastExecutionAt,
		&order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID,
		&order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt, &order.ExpiredAt, &order.ExpiryReason,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNoEligibleOrder
		}
		return nil, err
	}
	if order.ExpiryReason == nil {
		return nil, fmt.Errorf("expired order %s has no expiry reason", order.ID)
	}

	event := expiredEvent(order, *order.ExpiryReason)
	if err := insertEvent(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := insertOutbox(ctx, tx, event); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *Repository) DueStats(ctx context.Context, now time.Time) (int64, time.Duration, error) {
	var count int64
	var oldest *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*), MIN(expires_at)
		FROM orders
		WHERE expires_at IS NOT NULL
		  AND expires_at <= $1
		  AND remaining_quantity > 0
		  AND status IN ($2, $3)
		  AND time_in_force IN ($4, $5)
	`, now, domain.StatusAccepted, domain.StatusPartiallyFilled, domain.TimeInForceDay, domain.TimeInForceGTD).Scan(&count, &oldest)
	if err != nil {
		return 0, 0, err
	}
	if oldest == nil {
		return count, 0, nil
	}
	return count, now.Sub(*oldest), nil
}

func expiredEvent(order domain.Order, reason string) domain.OrderEvent {
	return domain.OrderEvent{
		EventID:           uuid.NewString(),
		EventType:         "order.expired",
		EventVersion:      "1.0",
		TenantID:          order.TenantID,
		OrderID:           order.ID,
		UserID:            order.UserID,
		Symbol:            order.Symbol,
		Side:              order.Side,
		OrderType:         order.OrderType,
		TimeInForce:       order.TimeInForce,
		Quantity:          order.Quantity,
		FilledQuantity:    order.FilledQuantity,
		RemainingQuantity: order.RemainingQuantity,
		Status:            order.Status,
		FillPrice:         order.FillPrice,
		AverageFillPrice:  order.AverageFillPrice,
		Version:           order.Version,
		ExpiresAt:         order.ExpiresAt,
		ExpiredAt:         order.ExpiredAt,
		ExpiryReason:      reason,
		Source:            "order-expiry-worker",
		OccurredAt:        time.Now().UTC(),
		CorrelationID:     order.CorrelationID,
	}
}

func insertEvent(ctx context.Context, tx pgx.Tx, event domain.OrderEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO order_events (event_id, event_type, tenant_id, order_id, user_id, payload, correlation_id, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.EventID, event.EventType, event.TenantID, event.OrderID, event.UserID, payload, event.CorrelationID, event.OccurredAt)
	return err
}

func insertOutbox(ctx context.Context, tx pgx.Tx, event domain.OrderEvent) error {
	ctx, span := otel.Tracer("order-service").Start(ctx, "order_expiry.write_outbox")
	defer span.End()

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO order_outbox (tenant_id, aggregate_type, aggregate_id, event_id, event_type, topic, payload, correlation_id, traceparent)
		VALUES ($1, 'order', $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (event_id) DO NOTHING
	`, event.TenantID, event.OrderID, event.EventID, event.EventType, event.EventType, payload, event.CorrelationID, event.TraceParent)
	return err
}

func correlationID(orderID string, now time.Time) string {
	return "expiry:" + orderID + ":" + now.Format("20060102T150405.000000000Z")
}
