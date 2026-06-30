package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/matching"
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

func (r *OrderRepository) FindIdempotency(ctx context.Context, tenantID, userID, key string) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(tenant_id, 'default-tenant'), user_id, key, request_hash, order_id::text, created_at
		FROM idempotency_keys
		WHERE COALESCE(tenant_id, 'default-tenant') = $1 AND user_id = $2 AND key = $3
	`, tenantID, userID, key).Scan(&record.TenantID, &record.UserID, &record.Key, &record.RequestHash, &record.OrderID, &record.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.IdempotencyRecord{}, ErrNotFound
		}
		return domain.IdempotencyRecord{}, err
	}
	return record, nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order domain.Order, events []domain.OrderEvent, idempotencyKey, requestHash string) (domain.Order, []domain.OrderEvent, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Order{}, nil, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (tenant_id, user_id, symbol, side, order_type, quantity, filled_quantity, remaining_quantity, limit_price, stop_price, average_fill_price, time_in_force, expires_at, version, risk_decision_id, last_execution_at, status, fill_price, reject_reason, correlation_id, cancelled_at, filled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id::text, created_at, updated_at
	`, order.TenantID, order.UserID, order.Symbol, order.Side, order.OrderType, order.Quantity, order.FilledQuantity, order.RemainingQuantity, order.LimitPrice, order.StopPrice, order.AverageFillPrice, order.TimeInForce, order.ExpiresAt, order.Version, order.RiskDecisionID, order.LastExecutionAt, order.Status, order.FillPrice, order.RejectReason, order.CorrelationID, order.CancelledAt, order.FilledAt).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return domain.Order{}, nil, err
	}

	if order.Status == domain.StatusAccepted {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, order.TenantID+":"+order.Symbol); err != nil {
			return domain.Order{}, nil, err
		}
		var matchEvents []domain.OrderEvent
		order, matchEvents, err = r.matchAcceptedOrder(ctx, tx, order)
		if err != nil {
			return domain.Order{}, nil, err
		}
		events = append(events, matchEvents...)
	}

	for i := range events {
		if events[i].OrderID == "" {
			events[i].OrderID = order.ID
		}
		if err := insertEvent(ctx, tx, events[i]); err != nil {
			return domain.Order{}, nil, err
		}
		if err := insertOutbox(ctx, tx, events[i]); err != nil {
			return domain.Order{}, nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO idempotency_keys (tenant_id, user_id, key, request_hash, order_id)
		VALUES ($1, $2, $3, $4, $5)
	`, order.TenantID, order.UserID, idempotencyKey, requestHash, order.ID); err != nil {
		return domain.Order{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, nil, err
	}
	return order, events, nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, tenantID, id string) (domain.Order, error) {
	var order domain.Order
	err := r.db.QueryRow(ctx, `
		SELECT id::text, COALESCE(tenant_id, 'default-tenant'), user_id, symbol, side, order_type, quantity::float8, filled_quantity::float8, remaining_quantity::float8, limit_price::float8, stop_price::float8, average_fill_price::float8, time_in_force, expires_at, version, risk_decision_id::text, last_execution_at, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
		FROM orders
		WHERE id = $1 AND COALESCE(tenant_id, 'default-tenant') = $2
	`, id, tenantID).Scan(&order.ID, &order.TenantID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.FilledQuantity, &order.RemainingQuantity, &order.LimitPrice, &order.StopPrice, &order.AverageFillPrice, &order.TimeInForce, &order.ExpiresAt, &order.Version, &order.RiskDecisionID, &order.LastExecutionAt, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Order{}, ErrNotFound
		}
		return domain.Order{}, err
	}
	return order, nil
}

func (r *OrderRepository) ListOrders(ctx context.Context, tenantID, userID string, includeAll bool) ([]domain.Order, error) {
	query := `
		SELECT id::text, COALESCE(tenant_id, 'default-tenant'), user_id, symbol, side, order_type, quantity::float8, filled_quantity::float8, remaining_quantity::float8, limit_price::float8, stop_price::float8, average_fill_price::float8, time_in_force, expires_at, version, risk_decision_id::text, last_execution_at, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
		FROM orders
	`
	args := []any{tenantID}
	query += ` WHERE COALESCE(tenant_id, 'default-tenant') = $1`
	if !includeAll {
		query += ` AND user_id = $2`
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
		if err := rows.Scan(&order.ID, &order.TenantID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.FilledQuantity, &order.RemainingQuantity, &order.LimitPrice, &order.StopPrice, &order.AverageFillPrice, &order.TimeInForce, &order.ExpiresAt, &order.Version, &order.RiskDecisionID, &order.LastExecutionAt, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (r *OrderRepository) CancelOrder(ctx context.Context, tenantID, id, correlationID string, event domain.OrderEvent) (domain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback(ctx)

	var order domain.Order
	err = tx.QueryRow(ctx, `
		UPDATE orders
		SET status = $2, remaining_quantity = 0, cancelled_at = now(), updated_at = now(), correlation_id = $3, version = version + 1
		WHERE id = $1 AND status IN ($4, $5) AND COALESCE(tenant_id, 'default-tenant') = $6
		RETURNING id::text, COALESCE(tenant_id, 'default-tenant'), user_id, symbol, side, order_type, quantity::float8, filled_quantity::float8, remaining_quantity::float8, limit_price::float8, stop_price::float8, average_fill_price::float8, time_in_force, expires_at, version, risk_decision_id::text, last_execution_at, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
	`, id, domain.StatusCancelled, correlationID, domain.StatusAccepted, domain.StatusPartiallyFilled, tenantID).Scan(&order.ID, &order.TenantID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.FilledQuantity, &order.RemainingQuantity, &order.LimitPrice, &order.StopPrice, &order.AverageFillPrice, &order.TimeInForce, &order.ExpiresAt, &order.Version, &order.RiskDecisionID, &order.LastExecutionAt, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt)
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
	if err := insertOutbox(ctx, tx, event); err != nil {
		return domain.Order{}, err
	}
	return order, tx.Commit(ctx)
}

func (r *OrderRepository) AmendOrder(ctx context.Context, tenantID, id string, req domain.AmendOrderRequest, correlationID string) (domain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback(ctx)

	var order domain.Order
	err = tx.QueryRow(ctx, `
		UPDATE orders
		SET quantity = COALESCE($3, quantity),
		    remaining_quantity = COALESCE($3, quantity) - filled_quantity,
		    limit_price = $4,
		    stop_price = $5,
		    expires_at = $6,
		    correlation_id = $7,
		    updated_at = now(),
		    version = version + 1
		WHERE id = $1
		  AND COALESCE(tenant_id, 'default-tenant') = $2
		  AND version = $8
		  AND status IN ($9, $10)
		  AND ($3::numeric IS NULL OR $3::numeric >= filled_quantity)
		RETURNING id::text, COALESCE(tenant_id, 'default-tenant'), user_id, symbol, side, order_type, quantity::float8, filled_quantity::float8, remaining_quantity::float8, limit_price::float8, stop_price::float8, average_fill_price::float8, time_in_force, expires_at, version, risk_decision_id::text, last_execution_at, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
	`, id, tenantID, req.Quantity, req.LimitPrice, req.StopPrice, req.ExpiresAt, correlationID, req.ExpectedVersion, domain.StatusAccepted, domain.StatusPartiallyFilled).Scan(&order.ID, &order.TenantID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.FilledQuantity, &order.RemainingQuantity, &order.LimitPrice, &order.StopPrice, &order.AverageFillPrice, &order.TimeInForce, &order.ExpiresAt, &order.Version, &order.RiskDecisionID, &order.LastExecutionAt, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Order{}, ErrNotFound
		}
		return domain.Order{}, err
	}
	event := newRepositoryEvent(order, "order.amended")
	if err := insertEvent(ctx, tx, event); err != nil {
		return domain.Order{}, err
	}
	if err := insertOutbox(ctx, tx, event); err != nil {
		return domain.Order{}, err
	}
	return order, tx.Commit(ctx)
}

func (r *OrderRepository) ListExecutionsForOrder(ctx context.Context, tenantID, orderID string) ([]domain.Execution, error) {
	return r.queryExecutions(ctx, `
		SELECT id::text, tenant_id, buy_order_id::text, sell_order_id::text, symbol, execution_quantity::float8, execution_price::float8, buyer_user_id, seller_user_id, COALESCE(correlation_id, ''), executed_at
		FROM order_executions
		WHERE tenant_id = $1 AND (buy_order_id = $2 OR sell_order_id = $2)
		ORDER BY executed_at DESC
		LIMIT 100
	`, tenantID, orderID)
}

func (r *OrderRepository) ListTrades(ctx context.Context, tenantID string, limit int) ([]domain.Execution, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	return r.queryExecutions(ctx, `
		SELECT id::text, tenant_id, buy_order_id::text, sell_order_id::text, symbol, execution_quantity::float8, execution_price::float8, buyer_user_id, seller_user_id, COALESCE(correlation_id, ''), executed_at
		FROM order_executions
		WHERE tenant_id = $1
		ORDER BY executed_at DESC
		LIMIT $2
	`, tenantID, limit)
}

func (r *OrderRepository) GetTrade(ctx context.Context, tenantID, id string) (domain.Execution, error) {
	execs, err := r.queryExecutions(ctx, `
		SELECT id::text, tenant_id, buy_order_id::text, sell_order_id::text, symbol, execution_quantity::float8, execution_price::float8, buyer_user_id, seller_user_id, COALESCE(correlation_id, ''), executed_at
		FROM order_executions
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	if err != nil {
		return domain.Execution{}, err
	}
	if len(execs) == 0 {
		return domain.Execution{}, ErrNotFound
	}
	return execs[0], nil
}

func (r *OrderRepository) queryExecutions(ctx context.Context, query string, args ...any) ([]domain.Execution, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var executions []domain.Execution
	for rows.Next() {
		var execution domain.Execution
		if err := rows.Scan(&execution.ID, &execution.TenantID, &execution.BuyOrderID, &execution.SellOrderID, &execution.Symbol, &execution.Quantity, &execution.Price, &execution.BuyerUserID, &execution.SellerUserID, &execution.CorrelationID, &execution.ExecutedAt); err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}
	return executions, rows.Err()
}

func (r *OrderRepository) OrderBookDepth(ctx context.Context, tenantID, symbol string, depth int) (map[string]any, error) {
	if depth <= 0 || depth > 100 {
		depth = 20
	}
	rows, err := r.db.Query(ctx, `
		SELECT side, limit_price::float8, SUM(remaining_quantity)::float8 AS quantity
		FROM orders
		WHERE tenant_id = $1 AND symbol = $2 AND status IN ($3, $4) AND remaining_quantity > 0 AND limit_price IS NOT NULL
		GROUP BY side, limit_price
		ORDER BY CASE WHEN side = 'BUY' THEN -limit_price ELSE limit_price END
	`, tenantID, symbol, domain.StatusAccepted, domain.StatusPartiallyFilled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bids, asks []map[string]float64
	for rows.Next() {
		var side string
		var price, quantity float64
		if err := rows.Scan(&side, &price, &quantity); err != nil {
			return nil, err
		}
		level := map[string]float64{"price": price, "quantity": quantity}
		if side == domain.SideBuy && len(bids) < depth {
			bids = append(bids, level)
		}
		if side == domain.SideSell && len(asks) < depth {
			asks = append(asks, level)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var bestBid, bestAsk, spread any
	if len(bids) > 0 {
		bestBid = bids[0]["price"]
	}
	if len(asks) > 0 {
		bestAsk = asks[0]["price"]
	}
	if len(bids) > 0 && len(asks) > 0 {
		spread = asks[0]["price"] - bids[0]["price"]
	}
	return map[string]any{
		"tenantId":    tenantID,
		"symbol":      symbol,
		"bestBid":     bestBid,
		"bestAsk":     bestAsk,
		"spread":      spread,
		"bids":        bids,
		"asks":        asks,
		"generatedAt": time.Now().UTC(),
	}, nil
}

type txQuerier interface {
	Exec(context.Context, string, ...any) (pgconnCommandTag, error)
}

type pgconnCommandTag interface{}

func (r *OrderRepository) matchAcceptedOrder(ctx context.Context, tx pgx.Tx, order domain.Order) (domain.Order, []domain.OrderEvent, error) {
	resting, err := loadRestingOrders(ctx, tx, order)
	if err != nil {
		return domain.Order{}, nil, err
	}
	result := matching.NewEngine().Match(matching.FromDomain(order), resting, time.Now().UTC())
	byID := map[string]*domain.Order{order.ID: &order}
	matching.ApplyToDomain(result.Incoming, &order)
	updateOrderLifecycleFromFill(&order, result.Incoming, result.Cancelled)
	events := lifecycleEvents(order)

	var changedResting []domain.Order
	for _, changed := range result.Resting {
		restingOrder, err := loadOrderForUpdate(ctx, tx, order.TenantID, changed.ID)
		if err != nil {
			return domain.Order{}, nil, err
		}
		beforeStatus := restingOrder.Status
		matching.ApplyToDomain(changed, &restingOrder)
		updateOrderLifecycleFromFill(&restingOrder, changed, false)
		byID[restingOrder.ID] = &restingOrder
		changedResting = append(changedResting, restingOrder)
		events = append(events, fillEventForTransition(restingOrder, beforeStatus))
	}

	for _, exec := range result.Executions {
		if err := insertExecution(ctx, tx, order.TenantID, order.CorrelationID, exec); err != nil {
			return domain.Order{}, nil, err
		}
		events = append(events, tradeExecutedEvent(order, exec))
	}
	for _, restingOrder := range changedResting {
		if err := persistOrderState(ctx, tx, restingOrder); err != nil {
			return domain.Order{}, nil, err
		}
	}

	if len(result.Executions) > 0 || result.Cancelled {
		if err := persistOrderState(ctx, tx, order); err != nil {
			return domain.Order{}, nil, err
		}
	} else if result.Rested {
		events = nil
	}
	_ = byID
	return order, compactEvents(events), nil
}

func loadRestingOrders(ctx context.Context, tx pgx.Tx, incoming domain.Order) ([]matching.Order, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(tenant_id, 'default-tenant'), user_id, symbol, side, order_type, quantity::float8, filled_quantity::float8, remaining_quantity::float8, limit_price::float8, COALESCE(time_in_force, 'DAY'), created_at
		FROM orders
		WHERE COALESCE(tenant_id, 'default-tenant') = $1
		  AND symbol = $2
		  AND side <> $3
		  AND status IN ($4, $5)
		  AND remaining_quantity > 0
		  AND order_type = $6
		FOR UPDATE
	`, incoming.TenantID, incoming.Symbol, incoming.Side, domain.StatusAccepted, domain.StatusPartiallyFilled, domain.OrderTypeLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []matching.Order
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(&order.ID, &order.TenantID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.FilledQuantity, &order.RemainingQuantity, &order.LimitPrice, &order.TimeInForce, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, matching.FromDomain(order))
	}
	return orders, rows.Err()
}

func loadOrderForUpdate(ctx context.Context, tx pgx.Tx, tenantID, id string) (domain.Order, error) {
	var order domain.Order
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(tenant_id, 'default-tenant'), user_id, symbol, side, order_type, quantity::float8, filled_quantity::float8, remaining_quantity::float8, limit_price::float8, stop_price::float8, average_fill_price::float8, time_in_force, expires_at, version, risk_decision_id::text, last_execution_at, status, fill_price::float8, reject_reason, COALESCE(correlation_id, ''), created_at, updated_at, cancelled_at, filled_at
		FROM orders
		WHERE id = $1 AND COALESCE(tenant_id, 'default-tenant') = $2
		FOR UPDATE
	`, id, tenantID).Scan(&order.ID, &order.TenantID, &order.UserID, &order.Symbol, &order.Side, &order.OrderType, &order.Quantity, &order.FilledQuantity, &order.RemainingQuantity, &order.LimitPrice, &order.StopPrice, &order.AverageFillPrice, &order.TimeInForce, &order.ExpiresAt, &order.Version, &order.RiskDecisionID, &order.LastExecutionAt, &order.Status, &order.FillPrice, &order.RejectReason, &order.CorrelationID, &order.CreatedAt, &order.UpdatedAt, &order.CancelledAt, &order.FilledAt)
	return order, err
}

func updateOrderLifecycleFromFill(order *domain.Order, matched matching.Order, cancelled bool) {
	now := time.Now().UTC()
	order.FilledQuantity = matching.RatFloat(matched.FilledQuantity)
	if cancelled {
		order.RemainingQuantity = 0
		if order.FilledQuantity > 0 {
			order.Status = domain.StatusCancelled
		} else {
			order.Status = domain.StatusCancelled
		}
		order.CancelledAt = &now
		return
	}
	order.RemainingQuantity = matching.RatFloat(matched.RemainingQuantity)
	if order.FilledQuantity > 0 {
		if order.RemainingQuantity == 0 {
			order.Status = domain.StatusFilled
			order.FilledAt = &now
		} else {
			order.Status = domain.StatusPartiallyFilled
		}
		order.LastExecutionAt = &now
	}
}

func persistOrderState(ctx context.Context, tx pgx.Tx, order domain.Order) error {
	_, err := tx.Exec(ctx, `
		UPDATE orders
		SET filled_quantity = $2,
		    remaining_quantity = $3,
		    average_fill_price = (
		      SELECT CASE WHEN SUM(execution_quantity) > 0
		        THEN SUM(execution_quantity * execution_price) / SUM(execution_quantity)
		        ELSE NULL
		      END
		      FROM order_executions
		      WHERE tenant_id = $4 AND (buy_order_id = $1 OR sell_order_id = $1)
		    ),
		    fill_price = (
		      SELECT CASE WHEN SUM(execution_quantity) > 0
		        THEN SUM(execution_quantity * execution_price) / SUM(execution_quantity)
		        ELSE NULL
		      END
		      FROM order_executions
		      WHERE tenant_id = $4 AND (buy_order_id = $1 OR sell_order_id = $1)
		    ),
		    status = $5,
		    cancelled_at = $6,
		    filled_at = $7,
		    last_execution_at = $8,
		    updated_at = now(),
		    version = version + 1
		WHERE id = $1 AND COALESCE(tenant_id, 'default-tenant') = $4
	`, order.ID, order.FilledQuantity, order.RemainingQuantity, order.TenantID, order.Status, order.CancelledAt, order.FilledAt, order.LastExecutionAt)
	return err
}

func insertExecution(ctx context.Context, tx pgx.Tx, tenantID, correlationID string, exec matching.Execution) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO order_executions (tenant_id, buy_order_id, sell_order_id, symbol, execution_quantity, execution_price, buyer_user_id, seller_user_id, correlation_id, executed_at)
		VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7, $8, $9, $10)
	`, tenantID, exec.BuyOrderID, exec.SellOrderID, exec.Symbol, matching.RatString(exec.Quantity), matching.RatString(exec.Price), exec.BuyerUserID, exec.SellerUserID, correlationID, exec.ExecutedAt)
	return err
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

func lifecycleEvents(order domain.Order) []domain.OrderEvent {
	switch order.Status {
	case domain.StatusPartiallyFilled:
		return []domain.OrderEvent{newRepositoryEvent(order, "order.partially_filled")}
	case domain.StatusFilled:
		return []domain.OrderEvent{newRepositoryEvent(order, "order.filled")}
	case domain.StatusCancelled:
		return []domain.OrderEvent{newRepositoryEvent(order, "order.cancelled")}
	default:
		return nil
	}
}

func fillEventForTransition(order domain.Order, beforeStatus string) domain.OrderEvent {
	if order.Status == domain.StatusFilled && beforeStatus != domain.StatusFilled {
		return newRepositoryEvent(order, "order.filled")
	}
	if order.Status == domain.StatusPartiallyFilled {
		return newRepositoryEvent(order, "order.partially_filled")
	}
	return domain.OrderEvent{}
}

func tradeExecutedEvent(order domain.Order, exec matching.Execution) domain.OrderEvent {
	event := newRepositoryEvent(order, "trade.executed")
	event.Quantity = matching.RatFloat(exec.Quantity)
	price := matching.RatFloat(exec.Price)
	event.FillPrice = &price
	event.BuyOrderID = exec.BuyOrderID
	event.SellOrderID = exec.SellOrderID
	event.ExecutionQuantity = event.Quantity
	event.ExecutionPrice = &price
	return event
}

func newRepositoryEvent(order domain.Order, eventType string) domain.OrderEvent {
	return domain.OrderEvent{
		EventID:           uuid.NewString(),
		EventType:         eventType,
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
		Version:           order.Version + 1,
		OccurredAt:        time.Now().UTC(),
		CorrelationID:     order.CorrelationID,
	}
}

func compactEvents(events []domain.OrderEvent) []domain.OrderEvent {
	compacted := events[:0]
	for _, event := range events {
		if event.EventType != "" {
			compacted = append(compacted, event)
		}
	}
	return compacted
}
