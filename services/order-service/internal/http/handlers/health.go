package handlers

import (
	"context"
	"net/http"
	"time"

	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type HealthHandler struct {
	db           *pgxpool.Pool
	kafkaBrokers []string
}

func NewHealthHandler(db *pgxpool.Pool, kafkaBrokers []string) *HealthHandler {
	return &HealthHandler{db: db, kafkaBrokers: kafkaBrokers}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "order-service"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		httpmiddleware.WriteError(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	conn, err := kafka.DialContext(ctx, "tcp", h.kafkaBrokers[0])
	if err != nil {
		httpmiddleware.WriteError(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	_ = conn.Close()
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready", "service": "order-service"})
}
