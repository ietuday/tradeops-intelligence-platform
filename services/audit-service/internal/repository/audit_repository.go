package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("duplicate audit log")

type AuditRepository struct {
	db *pgxpool.Pool
}

type ListFilters struct {
	TenantID      string
	EventType     string
	ServiceName   string
	ActorUserID   string
	EntityType    string
	EntityID      string
	Action        string
	Severity      string
	CorrelationID string
	From          *time.Time
	To            *time.Time
	Limit         int
	Offset        int
}

type Summary struct {
	Total       int64            `json:"total"`
	Last24Hours int64            `json:"last24Hours"`
	ByService   map[string]int64 `json:"byService"`
	ByEventType map[string]int64 `json:"byEventType"`
	BySeverity  map[string]int64 `json:"bySeverity"`
	ByAction    map[string]int64 `json:"byAction"`
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) CreateAuditLog(ctx context.Context, log domain.AuditLog) (domain.AuditLog, error) {
	metadata, err := json.Marshal(nonNilMap(log.Metadata))
	if err != nil {
		return domain.AuditLog{}, err
	}
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	if log.Severity == "" {
		log.Severity = domain.SeverityInfo
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_logs (
			id, tenant_id, event_type, service_name, actor_user_id, actor_role, entity_type, entity_id,
			action, description, severity, correlation_id, ip_address, user_agent, metadata,
			source_event_key, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`, log.ID, defaultTenant(log.TenantID), log.EventType, log.ServiceName, log.ActorUserID, log.ActorRole, log.EntityType, log.EntityID,
		log.Action, log.Description, log.Severity, log.CorrelationID, log.IPAddress, log.UserAgent, metadata,
		log.SourceEventKey, log.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.AuditLog{}, ErrDuplicate
		}
		return domain.AuditLog{}, err
	}
	return log, nil
}

func (r *AuditRepository) GetAuditLogByID(ctx context.Context, tenantID, id string) (domain.AuditLog, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, COALESCE(tenant_id, 'default-tenant'), event_type, service_name, actor_user_id, actor_role, entity_type, entity_id,
			action, description, severity, correlation_id, ip_address, user_agent, metadata,
			source_event_key, created_at
		FROM audit_logs WHERE id = $1 AND COALESCE(tenant_id, 'default-tenant') = $2
	`, id, defaultTenant(tenantID))
	return scanAuditLog(row)
}

func (r *AuditRepository) ListAuditLogs(ctx context.Context, filters ListFilters) ([]domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(tenant_id, 'default-tenant'), event_type, service_name, actor_user_id, actor_role, entity_type, entity_id,
			action, description, severity, correlation_id, ip_address, user_agent, metadata,
			source_event_key, created_at
		FROM audit_logs
		WHERE COALESCE(tenant_id, 'default-tenant') = $1
		  AND ($2 = '' OR event_type = $2)
		  AND ($3 = '' OR service_name = $3)
		  AND ($4 = '' OR actor_user_id::text = $4)
		  AND ($5 = '' OR entity_type = $5)
		  AND ($6 = '' OR entity_id = $6)
		  AND ($7 = '' OR action = $7)
		  AND ($8 = '' OR severity = $8)
		  AND ($9 = '' OR correlation_id = $9)
		  AND ($10::timestamptz IS NULL OR created_at >= $10)
		  AND ($11::timestamptz IS NULL OR created_at <= $11)
		ORDER BY created_at DESC
		LIMIT $12 OFFSET $13
	`, defaultTenant(filters.TenantID), filters.EventType, filters.ServiceName, filters.ActorUserID, filters.EntityType, filters.EntityID,
		filters.Action, filters.Severity, filters.CorrelationID, filters.From, filters.To, filters.Limit, filters.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []domain.AuditLog
	for rows.Next() {
		log, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *AuditRepository) GetAuditSummary(ctx context.Context, filters ListFilters) (Summary, error) {
	summary := Summary{
		ByService:   map[string]int64{},
		ByEventType: map[string]int64{},
		BySeverity:  map[string]int64{},
		ByAction:    map[string]int64{},
	}
	tenantID := defaultTenant(filters.TenantID)
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE COALESCE(tenant_id, 'default-tenant') = $1`, tenantID).Scan(&summary.Total); err != nil {
		return summary, err
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE COALESCE(tenant_id, 'default-tenant') = $1 AND created_at >= now() - interval '24 hours'`, tenantID).Scan(&summary.Last24Hours); err != nil {
		return summary, err
	}
	if err := r.fillCountMap(ctx, `SELECT service_name, count(*) FROM audit_logs WHERE COALESCE(tenant_id, 'default-tenant') = $1 GROUP BY service_name`, tenantID, summary.ByService); err != nil {
		return summary, err
	}
	if err := r.fillCountMap(ctx, `SELECT event_type, count(*) FROM audit_logs WHERE COALESCE(tenant_id, 'default-tenant') = $1 GROUP BY event_type`, tenantID, summary.ByEventType); err != nil {
		return summary, err
	}
	if err := r.fillCountMap(ctx, `SELECT severity, count(*) FROM audit_logs WHERE COALESCE(tenant_id, 'default-tenant') = $1 GROUP BY severity`, tenantID, summary.BySeverity); err != nil {
		return summary, err
	}
	if err := r.fillCountMap(ctx, `SELECT action, count(*) FROM audit_logs WHERE COALESCE(tenant_id, 'default-tenant') = $1 GROUP BY action`, tenantID, summary.ByAction); err != nil {
		return summary, err
	}
	return summary, nil
}

func (r *AuditRepository) ExportAuditLogs(ctx context.Context, filters ListFilters) ([]domain.AuditLog, error) {
	if filters.Limit <= 0 || filters.Limit > 1000 {
		filters.Limit = 1000
	}
	return r.ListAuditLogs(ctx, filters)
}

func (r *AuditRepository) CreateExportRequest(ctx context.Context, request domain.ExportRequest) (domain.ExportRequest, error) {
	if request.ID == "" {
		request.ID = uuid.NewString()
	}
	if request.Status == "" {
		request.Status = "COMPLETED"
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now().UTC()
	}
	filters, err := json.Marshal(nonNilMap(request.Filters))
	if err != nil {
		return domain.ExportRequest{}, err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_export_requests (id, tenant_id, requested_by, filters, status, file_name, record_count, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, request.ID, defaultTenant(request.TenantID), request.RequestedBy, filters, request.Status, request.FileName, request.RecordCount, request.CreatedAt)
	return request, err
}

func (r *AuditRepository) fillCountMap(ctx context.Context, query, tenantID string, target map[string]int64) error {
	rows, err := r.db.Query(ctx, query, tenantID)
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
		target[key] = count
	}
	return rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAuditLog(row scanner) (domain.AuditLog, error) {
	var log domain.AuditLog
	var metadata []byte
	err := row.Scan(&log.ID, &log.TenantID, &log.EventType, &log.ServiceName, &log.ActorUserID, &log.ActorRole,
		&log.EntityType, &log.EntityID, &log.Action, &log.Description, &log.Severity,
		&log.CorrelationID, &log.IPAddress, &log.UserAgent, &metadata, &log.SourceEventKey, &log.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuditLog{}, ErrNotFound
	}
	if err != nil {
		return domain.AuditLog{}, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &log.Metadata); err != nil {
			return domain.AuditLog{}, err
		}
	}
	if log.Metadata == nil {
		log.Metadata = map[string]any{}
	}
	return log, nil
}

func defaultTenant(value string) string {
	if value == "" {
		return "default-tenant"
	}
	return value
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
