package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrInvalidTransition = errors.New("invalid alert status transition")

type AlertRepository struct {
	db *pgxpool.Pool
}

type AlertFilters struct {
	Status    string
	Severity  string
	AlertType string
	UserID    string
	Symbol    string
	Limit     int
	Offset    int
}

type Summary struct {
	OpenAlerts         int64            `json:"openAlerts"`
	AlertsBySeverity   map[string]int64 `json:"alertsBySeverity"`
	AlertsByType       map[string]int64 `json:"alertsByType"`
	CreatedLast24Hours int64            `json:"createdLast24Hours"`
}

func NewAlertRepository(db *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) CreateAlert(ctx context.Context, alert domain.Alert) (domain.Alert, error) {
	metadata, err := json.Marshal(alert.Metadata)
	if err != nil {
		return domain.Alert{}, err
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO surveillance_alerts (id, alert_type, severity, entity_type, entity_id, user_id, symbol, description, status, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)
		RETURNING id::text, alert_type, severity, entity_type, entity_id, user_id::text, symbol, description, status, metadata, created_at, acknowledged_at, resolved_at, dismissed_at
	`, alert.ID, alert.AlertType, alert.Severity, alert.EntityType, alert.EntityID, alert.UserID, alert.Symbol, alert.Description, alert.Status, string(metadata), alert.CreatedAt).Scan(
		&alert.ID, &alert.AlertType, &alert.Severity, &alert.EntityType, &alert.EntityID, &alert.UserID, &alert.Symbol, &alert.Description, &alert.Status, &metadata, &alert.CreatedAt, &alert.AcknowledgedAt, &alert.ResolvedAt, &alert.DismissedAt,
	)
	if err != nil {
		return domain.Alert{}, err
	}
	alert.Metadata = decodeMetadata(metadata)
	return alert, nil
}

func (r *AlertRepository) DuplicateAlertExists(ctx context.Context, alert domain.Alert) (bool, error) {
	sourceTopic := ""
	if value, ok := alert.Metadata["sourceTopic"].(string); ok {
		sourceTopic = value
	}
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM surveillance_alerts
			WHERE alert_type = $1
			  AND entity_id = $2
			  AND COALESCE(metadata->>'sourceTopic', '') = $3
		)
	`, alert.AlertType, alert.EntityID, sourceTopic).Scan(&exists)
	return exists, err
}

func (r *AlertRepository) ListAlerts(ctx context.Context, filters AlertFilters) ([]domain.Alert, error) {
	where, args := buildAlertFilters(filters)
	args = append(args, filters.Limit, filters.Offset)
	query := fmt.Sprintf(`
		SELECT id::text, alert_type, severity, entity_type, entity_id, user_id::text, symbol, description, status, metadata, created_at, acknowledged_at, resolved_at, dismissed_at
		FROM surveillance_alerts
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlerts(rows)
}

func (r *AlertRepository) GetAlert(ctx context.Context, id string) (domain.Alert, error) {
	var alert domain.Alert
	var metadata []byte
	err := r.db.QueryRow(ctx, `
		SELECT id::text, alert_type, severity, entity_type, entity_id, user_id::text, symbol, description, status, metadata, created_at, acknowledged_at, resolved_at, dismissed_at
		FROM surveillance_alerts
		WHERE id = $1
	`, id).Scan(&alert.ID, &alert.AlertType, &alert.Severity, &alert.EntityType, &alert.EntityID, &alert.UserID, &alert.Symbol, &alert.Description, &alert.Status, &metadata, &alert.CreatedAt, &alert.AcknowledgedAt, &alert.ResolvedAt, &alert.DismissedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Alert{}, ErrNotFound
	}
	if err != nil {
		return domain.Alert{}, err
	}
	alert.Metadata = decodeMetadata(metadata)
	return alert, nil
}

func (r *AlertRepository) UpdateStatus(ctx context.Context, id, status string) (domain.Alert, error) {
	column := map[string]string{
		domain.StatusAcknowledged: "acknowledged_at",
		domain.StatusResolved:     "resolved_at",
		domain.StatusDismissed:    "dismissed_at",
	}[status]
	if column == "" {
		return domain.Alert{}, ErrInvalidTransition
	}
	guard := ""
	if status == domain.StatusAcknowledged {
		guard = "AND status = 'OPEN'"
	}
	var alert domain.Alert
	var metadata []byte
	query := fmt.Sprintf(`
		UPDATE surveillance_alerts
		SET status = $2, %s = now()
		WHERE id = $1 %s
		RETURNING id::text, alert_type, severity, entity_type, entity_id, user_id::text, symbol, description, status, metadata, created_at, acknowledged_at, resolved_at, dismissed_at
	`, column, guard)
	err := r.db.QueryRow(ctx, query, id, status).Scan(&alert.ID, &alert.AlertType, &alert.Severity, &alert.EntityType, &alert.EntityID, &alert.UserID, &alert.Symbol, &alert.Description, &alert.Status, &metadata, &alert.CreatedAt, &alert.AcknowledgedAt, &alert.ResolvedAt, &alert.DismissedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.GetAlert(ctx, id); errors.Is(getErr, ErrNotFound) {
			return domain.Alert{}, ErrNotFound
		}
		return domain.Alert{}, ErrInvalidTransition
	}
	if err != nil {
		return domain.Alert{}, err
	}
	alert.Metadata = decodeMetadata(metadata)
	return alert, nil
}

func (r *AlertRepository) SaveExecution(ctx context.Context, execution domain.RuleExecution) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO surveillance_rule_executions (id, rule_name, source_topic, entity_id, matched, execution_time_ms, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, execution.ID, execution.RuleName, execution.SourceTopic, execution.EntityID, execution.Matched, execution.ExecutionTimeMS, execution.ErrorMessage)
	return err
}

func (r *AlertRepository) Summary(ctx context.Context) (Summary, error) {
	summary := Summary{AlertsBySeverity: map[string]int64{}, AlertsByType: map[string]int64{}}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM surveillance_alerts WHERE status = 'OPEN'`).Scan(&summary.OpenAlerts); err != nil {
		return summary, err
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM surveillance_alerts WHERE created_at >= now() - interval '24 hours'`).Scan(&summary.CreatedLast24Hours); err != nil {
		return summary, err
	}
	if err := scanCounts(ctx, r.db, `SELECT severity, COUNT(*) FROM surveillance_alerts GROUP BY severity`, summary.AlertsBySeverity); err != nil {
		return summary, err
	}
	if err := scanCounts(ctx, r.db, `SELECT alert_type, COUNT(*) FROM surveillance_alerts GROUP BY alert_type`, summary.AlertsByType); err != nil {
		return summary, err
	}
	return summary, nil
}

func buildAlertFilters(filters AlertFilters) (string, []any) {
	var clauses []string
	var args []any
	add := func(column, value string) {
		if value == "" {
			return
		}
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	add("status", strings.ToUpper(filters.Status))
	add("severity", strings.ToUpper(filters.Severity))
	add("alert_type", filters.AlertType)
	add("user_id::text", filters.UserID)
	add("symbol", strings.ToUpper(filters.Symbol))
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func scanAlerts(rows pgx.Rows) ([]domain.Alert, error) {
	var alerts []domain.Alert
	for rows.Next() {
		var alert domain.Alert
		var metadata []byte
		if err := rows.Scan(&alert.ID, &alert.AlertType, &alert.Severity, &alert.EntityType, &alert.EntityID, &alert.UserID, &alert.Symbol, &alert.Description, &alert.Status, &metadata, &alert.CreatedAt, &alert.AcknowledgedAt, &alert.ResolvedAt, &alert.DismissedAt); err != nil {
			return nil, err
		}
		alert.Metadata = decodeMetadata(metadata)
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

func decodeMetadata(data []byte) map[string]any {
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil || metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func scanCounts(ctx context.Context, db *pgxpool.Pool, query string, counts map[string]int64) error {
	rows, err := db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return err
		}
		counts[key] = count
	}
	return rows.Err()
}
