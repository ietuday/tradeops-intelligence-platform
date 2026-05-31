package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Record(ctx context.Context, userID *string, action, correlationID, ipAddress, userAgent string) {
	_, _ = r.db.Exec(ctx, `
		INSERT INTO audit_logs (user_id, action, correlation_id, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, action, correlationID, ipAddress, userAgent)
}
