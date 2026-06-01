package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicateEvent = errors.New("duplicate event")
var ErrInsufficientHoldings = errors.New("insufficient holdings")

type PortfolioRepository struct {
	db *pgxpool.Pool
}

type UpdateResult struct {
	Portfolio domain.Portfolio
	Holdings  []domain.Holding
	Snapshot  domain.Snapshot
	Duplicate bool
}

func NewPortfolioRepository(db *pgxpool.Pool) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

func (r *PortfolioRepository) ApplyFilledOrder(ctx context.Context, event domain.OrderFilledEvent, initialCash float64) (UpdateResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM processed_order_events WHERE event_id = $1)`, event.EventID).Scan(&exists); err != nil {
		return UpdateResult{}, err
	}
	if exists {
		return UpdateResult{Duplicate: true}, tx.Commit(ctx)
	}

	portfolioID, err := ensurePortfolio(ctx, tx, event.UserID, initialCash)
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
		INSERT INTO processed_order_events (event_id, order_id, user_id)
		VALUES ($1, $2, $3)
	`, event.EventID, event.OrderID, event.UserID); err != nil {
		return UpdateResult{}, err
	}

	portfolio, err := readPortfolio(ctx, tx, event.UserID)
	if err != nil {
		return UpdateResult{}, err
	}
	holdings, err := readHoldings(ctx, tx, event.UserID)
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

func (r *PortfolioRepository) GetPortfolio(ctx context.Context, userID string, initialCash float64) (domain.Portfolio, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Portfolio{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := ensurePortfolio(ctx, tx, userID, initialCash); err != nil {
		return domain.Portfolio{}, err
	}
	portfolio, err := readPortfolio(ctx, tx, userID)
	if err != nil {
		return domain.Portfolio{}, err
	}
	return portfolio, tx.Commit(ctx)
}

func (r *PortfolioRepository) GetHoldings(ctx context.Context, userID string) ([]domain.Holding, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, portfolio_id::text, user_id, symbol, quantity::float8, average_buy_price::float8, updated_at
		FROM portfolio_holdings
		WHERE user_id = $1 AND quantity > 0
		ORDER BY symbol
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHoldings(rows)
}

func (r *PortfolioRepository) GetSnapshots(ctx context.Context, userID string) ([]domain.Snapshot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, portfolio_id::text, user_id, cash_balance::float8, holdings_value::float8, total_value::float8, realized_pnl::float8, created_at
		FROM portfolio_snapshots
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []domain.Snapshot
	for rows.Next() {
		var snapshot domain.Snapshot
		if err := rows.Scan(&snapshot.ID, &snapshot.PortfolioID, &snapshot.UserID, &snapshot.CashBalance, &snapshot.HoldingsValue, &snapshot.TotalValue, &snapshot.RealizedPnL, &snapshot.CreatedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (r *PortfolioRepository) GetRealizedPnL(ctx context.Context, userID string) (float64, []map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT order_id, symbol, quantity::float8, fill_price::float8, average_buy_price::float8, realized_pnl::float8, occurred_at
		FROM realized_pnl_events
		WHERE user_id = $1
		ORDER BY occurred_at DESC
		LIMIT 100
	`, userID)
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

func ensurePortfolio(ctx context.Context, tx pgx.Tx, userID string, initialCash float64) (string, error) {
	var portfolioID string
	err := tx.QueryRow(ctx, `
		INSERT INTO portfolios (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = portfolios.updated_at
		RETURNING id::text
	`, userID).Scan(&portfolioID)
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
		INSERT INTO portfolio_holdings (portfolio_id, user_id, symbol, quantity, average_buy_price)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (portfolio_id, symbol) DO UPDATE
		SET average_buy_price = ((portfolio_holdings.quantity * portfolio_holdings.average_buy_price) + ($4 * $5)) / NULLIF(portfolio_holdings.quantity + $4, 0),
		    quantity = portfolio_holdings.quantity + $4,
		    updated_at = now()
	`, portfolioID, event.UserID, event.Symbol, event.Quantity, fillPrice)
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
		INSERT INTO realized_pnl_events (portfolio_id, user_id, order_id, symbol, quantity, fill_price, average_buy_price, realized_pnl, occurred_at, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, portfolioID, event.UserID, event.OrderID, event.Symbol, event.Quantity, fillPrice, averageBuyPrice, realized, event.OccurredAt, event.CorrelationID)
	return err
}

func readPortfolio(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string) (domain.Portfolio, error) {
	var portfolio domain.Portfolio
	err := q.QueryRow(ctx, `
		SELECT p.id::text, p.user_id, cb.cash_balance::float8, cb.realized_pnl::float8,
		       cb.cash_balance::float8 + COALESCE(SUM(ph.quantity * ph.average_buy_price), 0)::float8 AS total_value,
		       p.created_at, p.updated_at
		FROM portfolios p
		JOIN cash_balances cb ON cb.portfolio_id = p.id
		LEFT JOIN portfolio_holdings ph ON ph.portfolio_id = p.id
		WHERE p.user_id = $1
		GROUP BY p.id, p.user_id, cb.cash_balance, cb.realized_pnl, p.created_at, p.updated_at
	`, userID).Scan(&portfolio.ID, &portfolio.UserID, &portfolio.CashBalance, &portfolio.RealizedPnL, &portfolio.TotalValue, &portfolio.CreatedAt, &portfolio.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Portfolio{}, ErrNotFound
		}
		return domain.Portfolio{}, err
	}
	return portfolio, nil
}

func readHoldings(ctx context.Context, tx pgx.Tx, userID string) ([]domain.Holding, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, portfolio_id::text, user_id, symbol, quantity::float8, average_buy_price::float8, updated_at
		FROM portfolio_holdings
		WHERE user_id = $1 AND quantity > 0
		ORDER BY symbol
	`, userID)
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
		INSERT INTO portfolio_snapshots (portfolio_id, user_id, cash_balance, holdings_value, total_value, realized_pnl)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, portfolio_id::text, user_id, cash_balance::float8, holdings_value::float8, total_value::float8, realized_pnl::float8, created_at
	`, portfolio.ID, portfolio.UserID, portfolio.CashBalance, holdingsValue, portfolio.CashBalance+holdingsValue, portfolio.RealizedPnL).
		Scan(&snapshot.ID, &snapshot.PortfolioID, &snapshot.UserID, &snapshot.CashBalance, &snapshot.HoldingsValue, &snapshot.TotalValue, &snapshot.RealizedPnL, &snapshot.CreatedAt)
	return snapshot, err
}
