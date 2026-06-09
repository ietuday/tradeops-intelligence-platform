package repository

import (
	"context"
	"errors"

	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, fullName string, roles []string) (domain.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)

	var user domain.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, tenant_id)
		VALUES ($1, $2, $3, 'default-tenant')
		RETURNING id::text, COALESCE(tenant_id, 'default-tenant'), email, password_hash, full_name, created_at, updated_at
	`, email, passwordHash, fullName).Scan(&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.FullName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, ErrConflict
		}
		return domain.User{}, err
	}

	for _, role := range roles {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE name = $2
			ON CONFLICT DO NOTHING
		`, user.ID, role); err != nil {
			return domain.User{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	user.Roles = roles
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.findOne(ctx, "u.email = $1", email)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	return r.findOne(ctx, "u.id = $1", id)
}

func (r *UserRepository) findOne(ctx context.Context, predicate string, arg string) (domain.User, error) {
	var user domain.User
	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, COALESCE(u.tenant_id, 'default-tenant'), u.email, u.password_hash, u.full_name, u.created_at, u.updated_at, r.name
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN roles r ON r.id = ur.role_id
		WHERE `+predicate+`
		ORDER BY r.name
	`, arg)
	if err != nil {
		return domain.User{}, err
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var role *string
		if err := rows.Scan(&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.FullName, &user.CreatedAt, &user.UpdatedAt, &role); err != nil {
			return domain.User{}, err
		}
		found = true
		if role != nil {
			user.Roles = append(user.Roles, *role)
		}
	}
	if err := rows.Err(); err != nil {
		return domain.User{}, err
	}
	if !found {
		return domain.User{}, ErrNotFound
	}
	return user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
