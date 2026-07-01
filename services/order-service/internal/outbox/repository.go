package outbox

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Claim(ctx context.Context, workerID string, batchSize, maxAttempts int, leaseDuration time.Duration) ([]Record, int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id::text, tenant_id, aggregate_type, aggregate_id::text, event_id::text, event_type, topic,
		       payload, COALESCE(correlation_id, ''), COALESCE(traceparent, ''), COALESCE(tracestate, ''),
		       status, attempt_count, available_at, created_at, COALESCE(locked_by, ''),
		       locked_at, lease_until, published_at, failed_at, COALESCE(last_error, '')
		FROM order_outbox
		WHERE published_at IS NULL
		  AND status IN ('pending', 'processing')
		  AND available_at <= NOW()
		  AND attempt_count < $1
		  AND (lease_until IS NULL OR lease_until < NOW())
		ORDER BY created_at ASC, id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $2
	`, maxAttempts, batchSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records, err := scanRecords(rows)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(records))
	recovered := int64(0)
	for _, record := range records {
		ids = append(ids, record.ID)
		if record.LeaseUntil != nil && record.LeaseUntil.Before(time.Now()) {
			recovered++
		}
	}
	if len(ids) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, 0, err
		}
		return nil, 0, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE order_outbox
		SET status = 'processing',
		    locked_by = $1,
		    locked_at = NOW(),
		    lease_until = NOW() + ($2::text)::interval
		WHERE id::text = ANY($3)
	`, workerID, leaseDuration.String(), ids); err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	for i := range records {
		records[i].Status = StatusProcessing
		records[i].LockedBy = workerID
	}
	return records, recovered, nil
}

func (r *Repository) MarkPublished(ctx context.Context, id, workerID string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE order_outbox
		SET status = 'published',
		    published_at = NOW(),
		    locked_by = NULL,
		    locked_at = NULL,
		    lease_until = NULL,
		    last_error = NULL
		WHERE id = $1
		  AND published_at IS NULL
		  AND locked_by = $2
	`, id, workerID)
	return tag.RowsAffected() > 0, err
}

func (r *Repository) MarkFailed(ctx context.Context, id, workerID string, nextAvailableAt time.Time, safeError string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE order_outbox
		SET status = 'pending',
		    attempt_count = attempt_count + 1,
		    available_at = $3,
		    last_error = $4,
		    locked_by = NULL,
		    locked_at = NULL,
		    lease_until = NULL
		WHERE id = $1
		  AND published_at IS NULL
		  AND locked_by = $2
	`, id, workerID, nextAvailableAt, safeError)
	return tag.RowsAffected() > 0, err
}

func (r *Repository) MarkTerminal(ctx context.Context, id, workerID, safeError string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE order_outbox
		SET status = 'failed',
		    attempt_count = attempt_count + 1,
		    failed_at = NOW(),
		    last_error = $3,
		    locked_by = NULL,
		    locked_at = NULL,
		    lease_until = NULL
		WHERE id = $1
		  AND published_at IS NULL
		  AND locked_by = $2
	`, id, workerID, safeError)
	return tag.RowsAffected() > 0, err
}

func (r *Repository) Snapshot(ctx context.Context) (Stats, error) {
	var stats Stats
	var oldest pgtype.Timestamptz
	err := r.db.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE published_at IS NULL AND status = 'pending' AND attempt_count >= 0),
		  COUNT(*) FILTER (WHERE published_at IS NULL AND status = 'processing'),
		  COUNT(*) FILTER (WHERE status = 'failed'),
		  MIN(created_at) FILTER (WHERE published_at IS NULL AND status = 'pending')
		FROM order_outbox
	`).Scan(&stats.Pending, &stats.Processing, &stats.Failed, &oldest)
	if err != nil {
		return Stats{}, err
	}
	if oldest.Valid {
		stats.OldestPendingAge = time.Since(oldest.Time)
	}
	return stats, nil
}

func scanRecords(rows pgx.Rows) ([]Record, error) {
	var records []Record
	for rows.Next() {
		var record Record
		var lockedAt, leaseUntil, publishedAt, failedAt pgtype.Timestamptz
		if err := rows.Scan(
			&record.ID,
			&record.TenantID,
			&record.AggregateType,
			&record.AggregateID,
			&record.EventID,
			&record.EventType,
			&record.Topic,
			&record.Payload,
			&record.CorrelationID,
			&record.TraceParent,
			&record.TraceState,
			&record.Status,
			&record.AttemptCount,
			&record.AvailableAt,
			&record.CreatedAt,
			&record.LockedBy,
			&lockedAt,
			&leaseUntil,
			&publishedAt,
			&failedAt,
			&record.LastError,
		); err != nil {
			return nil, err
		}
		record.LockedAt = timePtr(lockedAt)
		record.LeaseUntil = timePtr(leaseUntil)
		record.PublishedAt = timePtr(publishedAt)
		record.FailedAt = timePtr(failedAt)
		records = append(records, record)
	}
	return records, rows.Err()
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}
