package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	db *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "notification-service",
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not_ready",
			"checks": map[string]string{"postgres": "down"},
		})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ready",
		"checks": map[string]string{"postgres": "up"},
	})
}
