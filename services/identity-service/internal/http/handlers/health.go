package handlers

import (
	"context"
	"net/http"
	"time"

	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redisClient}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "identity-service"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		httpmiddleware.WriteError(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		httpmiddleware.WriteError(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready", "service": "identity-service"})
}
