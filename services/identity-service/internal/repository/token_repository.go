package repository

import (
	"context"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/security"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type RefreshTokenRepository struct {
	db     *pgxpool.Pool
	redis  *redis.Client
	secret []byte
}

func NewRefreshTokenRepository(db *pgxpool.Pool, redisClient *redis.Client, secret []byte) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db, redis: redisClient, secret: secret}
}

func (r *RefreshTokenRepository) Store(ctx context.Context, userID, token string, ttl time.Duration) error {
	hash := security.HashRefreshToken(token, r.secret)
	expiresAt := time.Now().UTC().Add(ttl)
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash, expiresAt)
	if err != nil {
		return err
	}
	return r.redis.Set(ctx, redisKey(hash), userID, ttl).Err()
}

func (r *RefreshTokenRepository) FindActive(ctx context.Context, token string) (domain.RefreshToken, error) {
	hash := security.HashRefreshToken(token, r.secret)
	if err := r.redis.Get(ctx, redisKey(hash)).Err(); err != nil {
		return domain.RefreshToken{}, ErrNotFound
	}
	var refresh domain.RefreshToken
	err := r.db.QueryRow(ctx, `
		SELECT id::text, user_id::text, token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
	`, hash).Scan(&refresh.ID, &refresh.UserID, &refresh.TokenHash, &refresh.ExpiresAt, &refresh.RevokedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.RefreshToken{}, ErrNotFound
		}
		return domain.RefreshToken{}, err
	}
	return refresh, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	hash := security.HashRefreshToken(token, r.secret)
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, hash)
	if err != nil {
		return err
	}
	return r.redis.Del(ctx, redisKey(hash)).Err()
}

func redisKey(hash string) string {
	return "identity:refresh:" + hash
}
