package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type Tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func Insert(ctx context.Context, tx Tx, event Event) error {
	headers, err := json.Marshal(event.Headers)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO portfolio_outbox_events (
		  event_id, tenant_id, aggregate_type, aggregate_id, event_type, event_version, topic, payload, headers
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (event_id) DO NOTHING
	`, event.EventID, event.TenantID, event.AggregateType, event.AggregateID, event.EventType, event.EventVersion, event.Topic, event.Payload, headers)
	return err
}

func (r *Repository) FetchPending(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, event_id, tenant_id, aggregate_type, aggregate_id, event_type, event_version,
		       topic, payload, headers, status, attempts, next_attempt_at, last_error, created_at, published_at
		FROM portfolio_outbox_events
		WHERE status IN ('pending', 'failed')
		  AND published_at IS NULL
		  AND next_attempt_at <= now()
		ORDER BY created_at ASC, id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, tx.Commit(ctx)
	}
	ids := make([]int64, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE portfolio_outbox_events
		SET status = 'publishing'
		WHERE id = ANY($1)
	`, ids); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	for i := range events {
		events[i].Status = StatusPublishing
	}
	return events, nil
}

func (r *Repository) MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE portfolio_outbox_events
		SET status = 'published',
		    published_at = $2,
		    last_error = NULL
		WHERE event_id = $1 AND published_at IS NULL
	`, eventID, publishedAt.UTC())
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, eventID string, publishErr error, nextAttemptAt time.Time, terminal bool) error {
	status := StatusPending
	if terminal {
		status = StatusFailed
	}
	_, err := r.db.Exec(ctx, `
		UPDATE portfolio_outbox_events
		SET status = $2,
		    attempts = attempts + 1,
		    next_attempt_at = $3,
		    last_error = $4
		WHERE event_id = $1 AND published_at IS NULL
	`, eventID, status, nextAttemptAt.UTC(), publishErr.Error())
	return err
}

func (r *Repository) Snapshot(ctx context.Context) (Status, error) {
	var status Status
	var oldest pgtype.Timestamptz
	var lastPublished pgtype.Timestamptz
	err := r.db.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE status = 'pending'),
		  COUNT(*) FILTER (WHERE status = 'publishing'),
		  COUNT(*) FILTER (WHERE status = 'published' AND published_at >= now() - interval '1 hour'),
		  COUNT(*) FILTER (WHERE status = 'failed'),
		  MIN(created_at) FILTER (WHERE status = 'pending'),
		  MAX(published_at)
		FROM portfolio_outbox_events
	`).Scan(&status.Pending, &status.Publishing, &status.PublishedLastHour, &status.Failed, &oldest, &lastPublished)
	if err != nil {
		return Status{}, err
	}
	if oldest.Valid {
		status.OldestPendingAge = time.Since(oldest.Time)
		status.OldestPendingAgeSeconds = status.OldestPendingAge.Seconds()
	}
	if lastPublished.Valid {
		t := lastPublished.Time
		status.LastPublishedAt = &t
	}
	return status, nil
}

func scanEvents(rows pgx.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var event Event
		var headers []byte
		var lastError *string
		var publishedAt pgtype.Timestamptz
		if err := rows.Scan(&event.ID, &event.EventID, &event.TenantID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.EventVersion, &event.Topic, &event.Payload, &headers, &event.Status, &event.Attempts, &event.NextAttemptAt, &lastError, &event.CreatedAt, &publishedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(headers, &event.Headers)
		event.LastError = lastError
		if publishedAt.Valid {
			t := publishedAt.Time
			event.PublishedAt = &t
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
