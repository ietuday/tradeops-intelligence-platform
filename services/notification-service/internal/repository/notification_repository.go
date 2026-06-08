package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrInvalidTransition = errors.New("invalid notification status transition")

type NotificationRepository struct {
	db *pgxpool.Pool
}

type ListFilters struct {
	UserID   string
	Status   string
	Channel  string
	Priority string
	Limit    int
	Offset   int
}

type Summary struct {
	Unread             int64            `json:"unread"`
	ByStatus           map[string]int64 `json:"byStatus"`
	ByChannel          map[string]int64 `json:"byChannel"`
	ByPriority         map[string]int64 `json:"byPriority"`
	CreatedLast24Hours int64            `json:"createdLast24Hours"`
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	metadata, err := json.Marshal(notification.Metadata)
	if err != nil {
		return domain.Notification{}, err
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO notifications (id, user_id, channel, priority, status, title, message, source, metadata, read_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12)
		RETURNING id::text, user_id::text, channel, priority, status, title, message, source, metadata, read_at, created_at, updated_at
	`, notification.ID, notification.UserID, notification.Channel, notification.Priority, notification.Status, notification.Title, notification.Message, notification.Source, string(metadata), notification.ReadAt, notification.CreatedAt, notification.UpdatedAt).Scan(
		&notification.ID, &notification.UserID, &notification.Channel, &notification.Priority, &notification.Status, &notification.Title, &notification.Message, &notification.Source, &metadata, &notification.ReadAt, &notification.CreatedAt, &notification.UpdatedAt,
	)
	if err != nil {
		return domain.Notification{}, err
	}
	notification.Metadata = decodeMetadata(metadata)
	return notification, nil
}

func (r *NotificationRepository) List(ctx context.Context, filters ListFilters) ([]domain.Notification, error) {
	where, args := buildFilters(filters)
	args = append(args, filters.Limit, filters.Offset)
	query := fmt.Sprintf(`
		SELECT id::text, user_id::text, channel, priority, status, title, message, source, metadata, read_at, created_at, updated_at
		FROM notifications
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotifications(rows)
}

func (r *NotificationRepository) Get(ctx context.Context, id string) (domain.Notification, error) {
	var notification domain.Notification
	var metadata []byte
	err := r.db.QueryRow(ctx, `
		SELECT id::text, user_id::text, channel, priority, status, title, message, source, metadata, read_at, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`, id).Scan(&notification.ID, &notification.UserID, &notification.Channel, &notification.Priority, &notification.Status, &notification.Title, &notification.Message, &notification.Source, &metadata, &notification.ReadAt, &notification.CreatedAt, &notification.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Notification{}, ErrNotFound
	}
	if err != nil {
		return domain.Notification{}, err
	}
	notification.Metadata = decodeMetadata(metadata)
	return notification, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id string) (domain.Notification, error) {
	return r.updateStatus(ctx, id, domain.StatusRead, true)
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id, status string) (domain.Notification, error) {
	return r.updateStatus(ctx, id, status, false)
}

func (r *NotificationRepository) RecordAttempt(ctx context.Context, attempt domain.DeliveryAttempt) (domain.DeliveryAttempt, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO notification_delivery_attempts (id, notification_id, channel, status, attempt_number, error_message, attempted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, notification_id::text, channel, status, attempt_number, error_message, attempted_at
	`, attempt.ID, attempt.NotificationID, attempt.Channel, attempt.Status, attempt.AttemptNumber, attempt.ErrorMessage, attempt.AttemptedAt).Scan(&attempt.ID, &attempt.NotificationID, &attempt.Channel, &attempt.Status, &attempt.AttemptNumber, &attempt.ErrorMessage, &attempt.AttemptedAt)
	return attempt, err
}

func (r *NotificationRepository) Retry(ctx context.Context, id string) (domain.Notification, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Notification{}, err
	}
	defer tx.Rollback(ctx)

	var notification domain.Notification
	var metadata []byte
	err = tx.QueryRow(ctx, `
		UPDATE notifications
		SET status = $2, updated_at = now()
		WHERE id = $1 AND status = 'FAILED'
		RETURNING id::text, user_id::text, channel, priority, status, title, message, source, metadata, read_at, created_at, updated_at
	`, id, domain.StatusRetrying).Scan(&notification.ID, &notification.UserID, &notification.Channel, &notification.Priority, &notification.Status, &notification.Title, &notification.Message, &notification.Source, &metadata, &notification.ReadAt, &notification.CreatedAt, &notification.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.Get(ctx, id); errors.Is(getErr, ErrNotFound) {
			return domain.Notification{}, ErrNotFound
		}
		return domain.Notification{}, ErrInvalidTransition
	}
	if err != nil {
		return domain.Notification{}, err
	}
	notification.Metadata = decodeMetadata(metadata)
	if _, err := tx.Exec(ctx, `
		INSERT INTO notification_delivery_attempts (id, notification_id, channel, status, attempt_number, error_message)
		VALUES (gen_random_uuid(), $1, $2, $3, COALESCE((SELECT MAX(attempt_number) + 1 FROM notification_delivery_attempts WHERE notification_id = $1), 1), $4)
	`, id, notification.Channel, domain.StatusRetrying, "manual retry requested; delivery implementation deferred to Phase 2"); err != nil {
		return domain.Notification{}, err
	}
	return notification, tx.Commit(ctx)
}

func (r *NotificationRepository) Preferences(ctx context.Context, userID string) (domain.Preferences, error) {
	var prefs domain.Preferences
	err := r.db.QueryRow(ctx, `
		INSERT INTO notification_preferences (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = notification_preferences.updated_at
		RETURNING user_id::text, in_app_enabled, webhook_enabled, email_enabled, webhook_url, email_address, min_priority, created_at, updated_at
	`, userID).Scan(&prefs.UserID, &prefs.InAppEnabled, &prefs.WebhookEnabled, &prefs.EmailEnabled, &prefs.WebhookURL, &prefs.EmailAddress, &prefs.MinPriority, &prefs.CreatedAt, &prefs.UpdatedAt)
	return prefs, err
}

func (r *NotificationRepository) UpdatePreferences(ctx context.Context, prefs domain.Preferences) (domain.Preferences, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO notification_preferences (user_id, in_app_enabled, webhook_enabled, email_enabled, webhook_url, email_address, min_priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE
		SET in_app_enabled = EXCLUDED.in_app_enabled,
		    webhook_enabled = EXCLUDED.webhook_enabled,
		    email_enabled = EXCLUDED.email_enabled,
		    webhook_url = EXCLUDED.webhook_url,
		    email_address = EXCLUDED.email_address,
		    min_priority = EXCLUDED.min_priority,
		    updated_at = now()
		RETURNING user_id::text, in_app_enabled, webhook_enabled, email_enabled, webhook_url, email_address, min_priority, created_at, updated_at
	`, prefs.UserID, prefs.InAppEnabled, prefs.WebhookEnabled, prefs.EmailEnabled, prefs.WebhookURL, prefs.EmailAddress, prefs.MinPriority).Scan(&prefs.UserID, &prefs.InAppEnabled, &prefs.WebhookEnabled, &prefs.EmailEnabled, &prefs.WebhookURL, &prefs.EmailAddress, &prefs.MinPriority, &prefs.CreatedAt, &prefs.UpdatedAt)
	return prefs, err
}

func (r *NotificationRepository) Summary(ctx context.Context, userID string) (Summary, error) {
	summary := Summary{ByStatus: map[string]int64{}, ByChannel: map[string]int64{}, ByPriority: map[string]int64{}}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND status <> 'READ'`, userID).Scan(&summary.Unread); err != nil {
		return summary, err
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND created_at >= now() - interval '24 hours'`, userID).Scan(&summary.CreatedLast24Hours); err != nil {
		return summary, err
	}
	if err := scanCounts(ctx, r.db, `SELECT status, COUNT(*) FROM notifications WHERE user_id = $1 GROUP BY status`, userID, summary.ByStatus); err != nil {
		return summary, err
	}
	if err := scanCounts(ctx, r.db, `SELECT channel, COUNT(*) FROM notifications WHERE user_id = $1 GROUP BY channel`, userID, summary.ByChannel); err != nil {
		return summary, err
	}
	if err := scanCounts(ctx, r.db, `SELECT priority, COUNT(*) FROM notifications WHERE user_id = $1 GROUP BY priority`, userID, summary.ByPriority); err != nil {
		return summary, err
	}
	return summary, nil
}

func (r *NotificationRepository) updateStatus(ctx context.Context, id, status string, markRead bool) (domain.Notification, error) {
	guard := ""
	if status == domain.StatusRetrying {
		guard = "AND status = 'FAILED'"
	}
	readExpr := "read_at"
	if markRead {
		readExpr = "now()"
	}
	var notification domain.Notification
	var metadata []byte
	query := fmt.Sprintf(`
		UPDATE notifications
		SET status = $2, read_at = %s, updated_at = now()
		WHERE id = $1 %s
		RETURNING id::text, user_id::text, channel, priority, status, title, message, source, metadata, read_at, created_at, updated_at
	`, readExpr, guard)
	err := r.db.QueryRow(ctx, query, id, status).Scan(&notification.ID, &notification.UserID, &notification.Channel, &notification.Priority, &notification.Status, &notification.Title, &notification.Message, &notification.Source, &metadata, &notification.ReadAt, &notification.CreatedAt, &notification.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := r.Get(ctx, id); errors.Is(getErr, ErrNotFound) {
			return domain.Notification{}, ErrNotFound
		}
		return domain.Notification{}, ErrInvalidTransition
	}
	if err != nil {
		return domain.Notification{}, err
	}
	notification.Metadata = decodeMetadata(metadata)
	return notification, nil
}

func (r *NotificationRepository) NextAttemptNumber(ctx context.Context, notificationID string) (int, error) {
	var next int
	err := r.db.QueryRow(ctx, `SELECT COALESCE(MAX(attempt_number) + 1, 1) FROM notification_delivery_attempts WHERE notification_id = $1`, notificationID).Scan(&next)
	return next, err
}

func buildFilters(filters ListFilters) (string, []any) {
	var clauses []string
	var args []any
	add := func(column, value string) {
		if value == "" {
			return
		}
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	add("user_id::text", filters.UserID)
	add("status", strings.ToUpper(filters.Status))
	add("channel", strings.ToUpper(filters.Channel))
	add("priority", strings.ToUpper(filters.Priority))
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func scanNotifications(rows pgx.Rows) ([]domain.Notification, error) {
	var notifications []domain.Notification
	for rows.Next() {
		var notification domain.Notification
		var metadata []byte
		if err := rows.Scan(&notification.ID, &notification.UserID, &notification.Channel, &notification.Priority, &notification.Status, &notification.Title, &notification.Message, &notification.Source, &metadata, &notification.ReadAt, &notification.CreatedAt, &notification.UpdatedAt); err != nil {
			return nil, err
		}
		notification.Metadata = decodeMetadata(metadata)
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

func decodeMetadata(data []byte) map[string]any {
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil || metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func scanCounts(ctx context.Context, db *pgxpool.Pool, query, userID string, counts map[string]int64) error {
	rows, err := db.Query(ctx, query, userID)
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
