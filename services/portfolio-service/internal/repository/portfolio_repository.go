package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/idempotency"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/outbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicateEvent = errors.New("duplicate event")
var ErrInsufficientHoldings = errors.New("insufficient holdings")
var ErrInsufficientCash = errors.New("insufficient cash")
var ErrInvalidExecution = errors.New("invalid trade execution")
var ErrPayloadConflict = errors.New("trade execution payload conflict")
var ErrSelfTrade = errors.New("self-trade execution")

type PortfolioRepository struct {
	db *pgxpool.Pool
}

type UpdateResult struct {
	Portfolio      domain.Portfolio
	Seller         domain.Portfolio
	Holdings       []domain.Holding
	Snapshot       domain.Snapshot
	PortfolioEvent domain.PortfolioEvent
	Duplicate      bool
}

func NewPortfolioRepository(db *pgxpool.Pool) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

func (r *PortfolioRepository) ApplyTradeExecution(ctx context.Context, event domain.TradeExecutedEvent, initialCash float64, payloadHash string, portfolioTopic string) (UpdateResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	defer tx.Rollback(ctx)

	tenantID := defaultTenant(event.TenantID)
	idempotencyKey := idempotency.BuildIdempotencyKey(event)
	var existingHash string
	err = tx.QueryRow(ctx, `
		SELECT payload_hash
		FROM portfolio_processed_events
		WHERE tenant_id = $1 AND idempotency_key = $2
	`, tenantID, idempotencyKey).Scan(&existingHash)
	if err == nil {
		if existingHash != "" && existingHash != payloadHash {
			return UpdateResult{}, ErrPayloadConflict
		}
		return UpdateResult{Duplicate: true}, tx.Commit(ctx)
	}
	if err != pgx.ErrNoRows {
		return UpdateResult{}, err
	}

	buyerPortfolioID, err := ensurePortfolio(ctx, tx, tenantID, event.BuyerUserID, initialCash)
	if err != nil {
		return UpdateResult{}, err
	}
	sellerPortfolioID, err := ensurePortfolio(ctx, tx, tenantID, event.SellerUserID, initialCash)
	if err != nil {
		return UpdateResult{}, err
	}

	portfolioIDs := uniqueSorted([]string{buyerPortfolioID, sellerPortfolioID})
	for _, portfolioID := range portfolioIDs {
		if _, err := tx.Exec(ctx, `SELECT 1 FROM cash_balances WHERE portfolio_id = $1 FOR UPDATE`, portfolioID); err != nil {
			return UpdateResult{}, err
		}
	}
	holdingKeys := []struct {
		portfolioID string
		userID      string
	}{
		{buyerPortfolioID, event.BuyerUserID},
		{sellerPortfolioID, event.SellerUserID},
	}
	sort.Slice(holdingKeys, func(i, j int) bool {
		if holdingKeys[i].portfolioID == holdingKeys[j].portfolioID {
			return event.Symbol < event.Symbol
		}
		return holdingKeys[i].portfolioID < holdingKeys[j].portfolioID
	})
	for _, key := range holdingKeys {
		if err := ensureHoldingForUpdate(ctx, tx, tenantID, key.portfolioID, key.userID, event.Symbol); err != nil {
			return UpdateResult{}, err
		}
	}

	if err := applyExecutionBuyer(ctx, tx, tenantID, buyerPortfolioID, event); err != nil {
		return UpdateResult{}, err
	}
	sellerRealized, err := applyExecutionSeller(ctx, tx, tenantID, sellerPortfolioID, event)
	if err != nil {
		return UpdateResult{}, err
	}
	if err := insertPortfolioTransaction(ctx, tx, tenantID, buyerPortfolioID, event.BuyerUserID, event.ExecutionID, event.BuyOrderID, event.Symbol, "BUY", event.ExecutionQuantity, event.ExecutionPrice, event.ExecutionQuantity*event.ExecutionPrice, 0, -(event.ExecutionQuantity * event.ExecutionPrice), event.Currency, 0, event.CorrelationID, event.OccurredAt); err != nil {
		return UpdateResult{}, err
	}
	if err := insertPortfolioTransaction(ctx, tx, tenantID, sellerPortfolioID, event.SellerUserID, event.ExecutionID, event.SellOrderID, event.Symbol, "SELL", event.ExecutionQuantity, event.ExecutionPrice, event.ExecutionQuantity*event.ExecutionPrice, 0, event.ExecutionQuantity*event.ExecutionPrice, event.Currency, sellerRealized, event.CorrelationID, event.OccurredAt); err != nil {
		return UpdateResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO portfolio_processed_events (
		  event_id, execution_id, tenant_id, event_type, event_version, source_service,
		  aggregate_id, idempotency_key, correlation_id, payload_hash, consumer_version, processed_at
		)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, $9, $10, 'v3.3.0', $11)
	`, event.EventID, event.ExecutionID, tenantID, event.EventType, event.EventVersion, sourceService(event.Source), event.ExecutionID, idempotencyKey, event.CorrelationID, payloadHash, time.Now().UTC()); err != nil {
		return UpdateResult{}, err
	}

	buyer, err := readPortfolio(ctx, tx, tenantID, event.BuyerUserID)
	if err != nil {
		return UpdateResult{}, err
	}
	seller, err := readPortfolio(ctx, tx, tenantID, event.SellerUserID)
	if err != nil {
		return UpdateResult{}, err
	}
	holdings, err := readHoldings(ctx, tx, tenantID, event.BuyerUserID)
	if err != nil {
		return UpdateResult{}, err
	}
	snapshot, err := createSnapshot(ctx, tx, buyer, holdings)
	if err != nil {
		return UpdateResult{}, err
	}
	portfolioEvent, err := buildPortfolioUpdatedEvent(event, buyer, holdings, -(event.ExecutionQuantity * event.ExecutionPrice))
	if err != nil {
		return UpdateResult{}, err
	}
	if err := insertPortfolioOutbox(ctx, tx, portfolioTopic, portfolioEvent); err != nil {
		return UpdateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{Portfolio: buyer, Seller: seller, Holdings: holdings, Snapshot: snapshot, PortfolioEvent: portfolioEvent}, nil
}

func buildPortfolioUpdatedEvent(event domain.TradeExecutedEvent, portfolio domain.Portfolio, holdings []domain.Holding, cashDelta float64) (domain.PortfolioEvent, error) {
	var positionQuantity, averagePrice string
	for _, holding := range holdings {
		if holding.Symbol == event.Symbol {
			positionQuantity = formatDecimal(holding.Quantity)
			averagePrice = formatDecimal(holding.AverageBuyPrice)
			break
		}
	}
	now := time.Now().UTC()
	return domain.PortfolioEvent{
		EventID:           uuid.NewString(),
		EventType:         "portfolio.updated",
		EventVersion:      "v1",
		TenantID:          portfolio.TenantID,
		PortfolioID:       portfolio.ID,
		UserID:            portfolio.UserID,
		AccountID:         portfolio.ID,
		Symbol:            event.Symbol,
		PositionQuantity:  positionQuantity,
		AveragePrice:      averagePrice,
		CashDelta:         formatDecimal(cashDelta),
		CashBalance:       portfolio.CashBalance,
		TotalValue:        portfolio.TotalValue,
		RealizedPnL:       portfolio.RealizedPnL,
		SourceEventID:     event.EventID,
		SourceExecutionID: event.ExecutionID,
		UpdatedAt:         now,
		OccurredAt:        now,
		CorrelationID:     event.CorrelationID,
	}, nil
}

func insertPortfolioOutbox(ctx context.Context, tx pgx.Tx, topic string, event domain.PortfolioEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return outbox.Insert(ctx, tx, outbox.Event{
		EventID:       event.EventID,
		TenantID:      event.TenantID,
		AggregateType: "portfolio",
		AggregateID:   event.PortfolioID,
		EventType:     event.EventType,
		EventVersion:  event.EventVersion,
		Topic:         topic,
		Payload:       payload,
		Headers: map[string]string{
			"eventType":     event.EventType,
			"eventVersion":  event.EventVersion,
			"correlationId": event.CorrelationID,
			"tenantId":      event.TenantID,
		},
	})
}

func (r *PortfolioRepository) ApplyFilledOrder(ctx context.Context, event domain.OrderFilledEvent, initialCash float64) (UpdateResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM processed_order_events WHERE tenant_id = $1 AND event_id = $2)`, defaultTenant(event.TenantID), event.EventID).Scan(&exists); err != nil {
		return UpdateResult{}, err
	}
	if exists {
		return UpdateResult{Duplicate: true}, tx.Commit(ctx)
	}

	portfolioID, err := ensurePortfolio(ctx, tx, defaultTenant(event.TenantID), event.UserID, initialCash)
	if err != nil {
		return UpdateResult{}, err
	}

	fillPrice := 0.0
	if event.FillPrice != nil {
		fillPrice = *event.FillPrice
	}
	if fillPrice <= 0 || event.Quantity <= 0 {
		return UpdateResult{}, errors.New("invalid filled order event")
	}

	if event.Side == "BUY" {
		if err := applyBuy(ctx, tx, portfolioID, event, fillPrice); err != nil {
			return UpdateResult{}, err
		}
	} else if event.Side == "SELL" {
		if err := applySell(ctx, tx, portfolioID, event, fillPrice); err != nil {
			return UpdateResult{}, err
		}
	} else {
		return UpdateResult{}, errors.New("unsupported side")
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO processed_order_events (event_id, tenant_id, order_id, user_id)
		VALUES ($1, $2, $3, $4)
	`, event.EventID, defaultTenant(event.TenantID), event.OrderID, event.UserID); err != nil {
		return UpdateResult{}, err
	}

	portfolio, err := readPortfolio(ctx, tx, defaultTenant(event.TenantID), event.UserID)
	if err != nil {
		return UpdateResult{}, err
	}
	holdings, err := readHoldings(ctx, tx, defaultTenant(event.TenantID), event.UserID)
	if err != nil {
		return UpdateResult{}, err
	}
	snapshot, err := createSnapshot(ctx, tx, portfolio, holdings)
	if err != nil {
		return UpdateResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{Portfolio: portfolio, Holdings: holdings, Snapshot: snapshot}, nil
}

func ensureHoldingForUpdate(ctx context.Context, tx pgx.Tx, tenantID, portfolioID, userID, symbol string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO portfolio_holdings (tenant_id, portfolio_id, user_id, symbol, quantity, average_buy_price)
		VALUES ($1, $2, $3, $4, 0, 0)
		ON CONFLICT (portfolio_id, symbol) DO NOTHING
	`, tenantID, portfolioID, userID, symbol)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT 1 FROM portfolio_holdings WHERE portfolio_id = $1 AND symbol = $2 FOR UPDATE`, portfolioID, symbol)
	return err
}

func applyExecutionBuyer(ctx context.Context, tx pgx.Tx, tenantID, portfolioID string, event domain.TradeExecutedEvent) error {
	gross := event.ExecutionQuantity * event.ExecutionPrice
	tag, err := tx.Exec(ctx, `
		UPDATE cash_balances
		SET cash_balance = cash_balance - $2::numeric, updated_at = now()
		WHERE portfolio_id = $1 AND cash_balance >= $2::numeric
	`, portfolioID, fmt.Sprintf("%.10f", gross))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInsufficientCash
	}
	_, err = tx.Exec(ctx, `
		UPDATE portfolio_holdings
		SET average_buy_price = CASE
		      WHEN quantity + $3::numeric = 0 THEN 0
		      ELSE ((quantity * average_buy_price) + ($3::numeric * $4::numeric)) / NULLIF(quantity + $3::numeric, 0)
		    END,
		    quantity = quantity + $3::numeric,
		    updated_at = now()
		WHERE tenant_id = $1 AND portfolio_id = $2 AND symbol = $5
	`, tenantID, portfolioID, fmt.Sprintf("%.10f", event.ExecutionQuantity), fmt.Sprintf("%.10f", event.ExecutionPrice), event.Symbol)
	return err
}

func applyExecutionSeller(ctx context.Context, tx pgx.Tx, tenantID, portfolioID string, event domain.TradeExecutedEvent) (float64, error) {
	var currentQty, averageBuyPrice float64
	if err := tx.QueryRow(ctx, `
		SELECT quantity::float8, average_buy_price::float8
		FROM portfolio_holdings
		WHERE tenant_id = $1 AND portfolio_id = $2 AND symbol = $3
		FOR UPDATE
	`, tenantID, portfolioID, event.Symbol).Scan(&currentQty, &averageBuyPrice); err != nil {
		return 0, err
	}
	if currentQty < event.ExecutionQuantity {
		return 0, ErrInsufficientHoldings
	}
	realized := (event.ExecutionPrice - averageBuyPrice) * event.ExecutionQuantity
	if _, err := tx.Exec(ctx, `
		UPDATE portfolio_holdings
		SET quantity = quantity - $4::numeric, updated_at = now()
		WHERE tenant_id = $1 AND portfolio_id = $2 AND symbol = $3
	`, tenantID, portfolioID, event.Symbol, fmt.Sprintf("%.10f", event.ExecutionQuantity)); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE cash_balances
		SET cash_balance = cash_balance + $2::numeric, realized_pnl = realized_pnl + $3::numeric, updated_at = now()
		WHERE portfolio_id = $1
	`, portfolioID, fmt.Sprintf("%.10f", event.ExecutionQuantity*event.ExecutionPrice), fmt.Sprintf("%.10f", realized)); err != nil {
		return 0, err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO realized_pnl_events (tenant_id, portfolio_id, user_id, order_id, symbol, quantity, fill_price, average_buy_price, realized_pnl, occurred_at, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8::numeric, $9::numeric, $10, $11)
	`, tenantID, portfolioID, event.SellerUserID, event.SellOrderID, event.Symbol, fmt.Sprintf("%.10f", event.ExecutionQuantity), fmt.Sprintf("%.10f", event.ExecutionPrice), fmt.Sprintf("%.10f", averageBuyPrice), fmt.Sprintf("%.10f", realized), event.OccurredAt, event.CorrelationID)
	return realized, err
}

func insertPortfolioTransaction(ctx context.Context, tx pgx.Tx, tenantID, portfolioID, userID, executionID, orderID, symbol, side string, quantity, price, gross, fee, netCash float64, currency string, realizedPnL float64, correlationID string, occurredAt time.Time) error {
	if currency == "" {
		currency = "USD"
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO portfolio_transactions (tenant_id, portfolio_id, user_id, execution_id, order_id, symbol, side, quantity, price, gross_amount, fee_amount, net_cash_amount, currency, realized_pnl, correlation_id, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::numeric, $9::numeric, $10::numeric, $11::numeric, $12::numeric, $13, $14::numeric, $15, $16)
		ON CONFLICT (tenant_id, portfolio_id, execution_id, side) DO NOTHING
	`, tenantID, portfolioID, userID, executionID, orderID, symbol, side, fmt.Sprintf("%.10f", quantity), fmt.Sprintf("%.10f", price), fmt.Sprintf("%.10f", gross), fmt.Sprintf("%.10f", fee), fmt.Sprintf("%.10f", netCash), currency, fmt.Sprintf("%.10f", realizedPnL), correlationID, occurredAt)
	return err
}

func uniqueSorted(values []string) []string {
	sort.Strings(values)
	out := values[:0]
	for _, value := range values {
		if value == "" || (len(out) > 0 && out[len(out)-1] == value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func PayloadHash(event domain.TradeExecutedEvent) string {
	normalized := strings.Join([]string{
		defaultTenant(event.TenantID),
		event.ExecutionID,
		event.BuyOrderID,
		event.SellOrderID,
		event.BuyerUserID,
		event.SellerUserID,
		strings.ToUpper(strings.TrimSpace(event.Symbol)),
		fmt.Sprintf("%.10f", event.ExecutionQuantity),
		fmt.Sprintf("%.10f", event.ExecutionPrice),
		strings.ToUpper(defaultCurrency(event.Currency)),
	}, "|")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func formatDecimal(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.8f", value), "0"), ".")
}

func sourceService(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "order-service"
	}
	return value
}

func defaultCurrency(value string) string {
	if strings.TrimSpace(value) == "" {
		return "USD"
	}
	return strings.ToUpper(strings.TrimSpace(value))
}

func (r *PortfolioRepository) GetPortfolio(ctx context.Context, tenantID, userID string, initialCash float64) (domain.Portfolio, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Portfolio{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := ensurePortfolio(ctx, tx, defaultTenant(tenantID), userID, initialCash); err != nil {
		return domain.Portfolio{}, err
	}
	portfolio, err := readPortfolio(ctx, tx, defaultTenant(tenantID), userID)
	if err != nil {
		return domain.Portfolio{}, err
	}
	return portfolio, tx.Commit(ctx)
}

func (r *PortfolioRepository) GetHoldings(ctx context.Context, tenantID, userID string) ([]domain.Holding, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, portfolio_id::text, user_id, symbol, quantity::float8, average_buy_price::float8, updated_at
		FROM portfolio_holdings
		WHERE COALESCE(tenant_id, 'default-tenant') = $1 AND user_id = $2 AND quantity > 0
		ORDER BY symbol
	`, defaultTenant(tenantID), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHoldings(rows)
}

func (r *PortfolioRepository) GetSnapshots(ctx context.Context, tenantID, userID string) ([]domain.Snapshot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, portfolio_id::text, COALESCE(tenant_id, 'default-tenant'), user_id, cash_balance::float8, holdings_value::float8, total_value::float8, realized_pnl::float8, created_at
		FROM portfolio_snapshots
		WHERE COALESCE(tenant_id, 'default-tenant') = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 50
	`, defaultTenant(tenantID), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []domain.Snapshot
	for rows.Next() {
		var snapshot domain.Snapshot
		if err := rows.Scan(&snapshot.ID, &snapshot.PortfolioID, &snapshot.TenantID, &snapshot.UserID, &snapshot.CashBalance, &snapshot.HoldingsValue, &snapshot.TotalValue, &snapshot.RealizedPnL, &snapshot.CreatedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (r *PortfolioRepository) GetRealizedPnL(ctx context.Context, tenantID, userID string) (float64, []map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT order_id, symbol, quantity::float8, fill_price::float8, average_buy_price::float8, realized_pnl::float8, occurred_at
		FROM realized_pnl_events
		WHERE COALESCE(tenant_id, 'default-tenant') = $1 AND user_id = $2
		ORDER BY occurred_at DESC
		LIMIT 100
	`, defaultTenant(tenantID), userID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	total := 0.0
	var events []map[string]any
	for rows.Next() {
		var orderID, symbol string
		var quantity, fillPrice, averageBuyPrice, realized float64
		var occurredAt time.Time
		if err := rows.Scan(&orderID, &symbol, &quantity, &fillPrice, &averageBuyPrice, &realized, &occurredAt); err != nil {
			return 0, nil, err
		}
		total += realized
		events = append(events, map[string]any{
			"orderId": orderID, "symbol": symbol, "quantity": quantity, "fillPrice": fillPrice,
			"averageBuyPrice": averageBuyPrice, "realizedPnl": realized, "occurredAt": occurredAt,
		})
	}
	return total, events, rows.Err()
}

func ensurePortfolio(ctx context.Context, tx pgx.Tx, tenantID, userID string, initialCash float64) (string, error) {
	var portfolioID string
	err := tx.QueryRow(ctx, `
		INSERT INTO portfolios (tenant_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (tenant_id, user_id) DO UPDATE SET updated_at = portfolios.updated_at
		RETURNING id::text
	`, defaultTenant(tenantID), userID).Scan(&portfolioID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO cash_balances (portfolio_id, cash_balance)
		VALUES ($1, $2)
		ON CONFLICT (portfolio_id) DO NOTHING
	`, portfolioID, initialCash)
	return portfolioID, err
}

func applyBuy(ctx context.Context, tx pgx.Tx, portfolioID string, event domain.OrderFilledEvent, fillPrice float64) error {
	cost := event.Quantity * fillPrice
	if _, err := tx.Exec(ctx, `
		UPDATE cash_balances
		SET cash_balance = cash_balance - $2, updated_at = now()
		WHERE portfolio_id = $1
	`, portfolioID, cost); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO portfolio_holdings (tenant_id, portfolio_id, user_id, symbol, quantity, average_buy_price)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (portfolio_id, symbol) DO UPDATE
		SET average_buy_price = ((portfolio_holdings.quantity * portfolio_holdings.average_buy_price) + ($5 * $6)) / NULLIF(portfolio_holdings.quantity + $5, 0),
		    quantity = portfolio_holdings.quantity + $5,
		    updated_at = now()
	`, defaultTenant(event.TenantID), portfolioID, event.UserID, event.Symbol, event.Quantity, fillPrice)
	return err
}

func applySell(ctx context.Context, tx pgx.Tx, portfolioID string, event domain.OrderFilledEvent, fillPrice float64) error {
	var currentQty, averageBuyPrice float64
	err := tx.QueryRow(ctx, `
		SELECT quantity::float8, average_buy_price::float8
		FROM portfolio_holdings
		WHERE portfolio_id = $1 AND symbol = $2
		FOR UPDATE
	`, portfolioID, event.Symbol).Scan(&currentQty, &averageBuyPrice)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrInsufficientHoldings
		}
		return err
	}
	if currentQty < event.Quantity {
		return ErrInsufficientHoldings
	}
	realized := (fillPrice - averageBuyPrice) * event.Quantity
	if _, err := tx.Exec(ctx, `
		UPDATE portfolio_holdings
		SET quantity = quantity - $3, updated_at = now()
		WHERE portfolio_id = $1 AND symbol = $2
	`, portfolioID, event.Symbol, event.Quantity); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE cash_balances
		SET cash_balance = cash_balance + $2, realized_pnl = realized_pnl + $3, updated_at = now()
		WHERE portfolio_id = $1
	`, portfolioID, event.Quantity*fillPrice, realized); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO realized_pnl_events (tenant_id, portfolio_id, user_id, order_id, symbol, quantity, fill_price, average_buy_price, realized_pnl, occurred_at, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, defaultTenant(event.TenantID), portfolioID, event.UserID, event.OrderID, event.Symbol, event.Quantity, fillPrice, averageBuyPrice, realized, event.OccurredAt, event.CorrelationID)
	return err
}

func readPortfolio(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenantID, userID string) (domain.Portfolio, error) {
	var portfolio domain.Portfolio
	err := q.QueryRow(ctx, `
		SELECT p.id::text, COALESCE(p.tenant_id, 'default-tenant'), p.user_id, cb.cash_balance::float8, cb.realized_pnl::float8,
		       cb.cash_balance::float8 + COALESCE(SUM(ph.quantity * ph.average_buy_price), 0)::float8 AS total_value,
		       p.created_at, p.updated_at
		FROM portfolios p
		JOIN cash_balances cb ON cb.portfolio_id = p.id
		LEFT JOIN portfolio_holdings ph ON ph.portfolio_id = p.id
		WHERE COALESCE(p.tenant_id, 'default-tenant') = $1 AND p.user_id = $2
		GROUP BY p.id, p.user_id, cb.cash_balance, cb.realized_pnl, p.created_at, p.updated_at
	`, defaultTenant(tenantID), userID).Scan(&portfolio.ID, &portfolio.TenantID, &portfolio.UserID, &portfolio.CashBalance, &portfolio.RealizedPnL, &portfolio.TotalValue, &portfolio.CreatedAt, &portfolio.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Portfolio{}, ErrNotFound
		}
		return domain.Portfolio{}, err
	}
	return portfolio, nil
}

func readHoldings(ctx context.Context, tx pgx.Tx, tenantID, userID string) ([]domain.Holding, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, portfolio_id::text, user_id, symbol, quantity::float8, average_buy_price::float8, updated_at
		FROM portfolio_holdings
		WHERE COALESCE(tenant_id, 'default-tenant') = $1 AND user_id = $2 AND quantity > 0
		ORDER BY symbol
	`, defaultTenant(tenantID), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHoldings(rows)
}

func scanHoldings(rows pgx.Rows) ([]domain.Holding, error) {
	var holdings []domain.Holding
	for rows.Next() {
		var holding domain.Holding
		if err := rows.Scan(&holding.ID, &holding.PortfolioID, &holding.UserID, &holding.Symbol, &holding.Quantity, &holding.AverageBuyPrice, &holding.UpdatedAt); err != nil {
			return nil, err
		}
		holdings = append(holdings, holding)
	}
	return holdings, rows.Err()
}

func createSnapshot(ctx context.Context, tx pgx.Tx, portfolio domain.Portfolio, holdings []domain.Holding) (domain.Snapshot, error) {
	holdingsValue := 0.0
	for _, holding := range holdings {
		holdingsValue += holding.Quantity * holding.AverageBuyPrice
	}
	var snapshot domain.Snapshot
	err := tx.QueryRow(ctx, `
		INSERT INTO portfolio_snapshots (tenant_id, portfolio_id, user_id, cash_balance, holdings_value, total_value, realized_pnl)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, portfolio_id::text, COALESCE(tenant_id, 'default-tenant'), user_id, cash_balance::float8, holdings_value::float8, total_value::float8, realized_pnl::float8, created_at
	`, defaultTenant(portfolio.TenantID), portfolio.ID, portfolio.UserID, portfolio.CashBalance, holdingsValue, portfolio.CashBalance+holdingsValue, portfolio.RealizedPnL).
		Scan(&snapshot.ID, &snapshot.PortfolioID, &snapshot.TenantID, &snapshot.UserID, &snapshot.CashBalance, &snapshot.HoldingsValue, &snapshot.TotalValue, &snapshot.RealizedPnL, &snapshot.CreatedAt)
	return snapshot, err
}

func defaultTenant(value string) string {
	if value == "" {
		return "default-tenant"
	}
	return value
}
