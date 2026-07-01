package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/expiry"
	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/outbox"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type OutboxStatusProvider interface {
	Status() outbox.Stats
}

type OrderExpiryStatusProvider interface {
	Status() expiry.Stats
}

type HealthHandler struct {
	db           *pgxpool.Pool
	kafkaBrokers []string
	outbox       OutboxStatusProvider
	expiry       OrderExpiryStatusProvider
}

func NewHealthHandler(db *pgxpool.Pool, kafkaBrokers []string, outbox OutboxStatusProvider, expiry OrderExpiryStatusProvider) *HealthHandler {
	return &HealthHandler{db: db, kafkaBrokers: kafkaBrokers, outbox: outbox, expiry: expiry}
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

func (h *HealthHandler) OutboxStatus(w http.ResponseWriter, _ *http.Request) {
	if h.outbox == nil {
		httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	status := h.outbox.Status()
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{
		"enabled":                 true,
		"pending":                 status.Pending,
		"processing":              status.Processing,
		"failed":                  status.Failed,
		"oldestPendingAgeSeconds": status.OldestPendingAge.Seconds(),
		"consecutiveErrors":       status.ConsecutiveErrors,
		"lastPoll":                status.LastPoll,
		"lastPublished":           status.LastPublished,
		"lastError":               status.LastError,
	})
}

func (h *HealthHandler) OrderExpiryStatus(w http.ResponseWriter, _ *http.Request) {
	if h.expiry == nil {
		httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	status := h.expiry.Status()
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{
		"enabled":                  status.Enabled,
		"running":                  status.Running,
		"timezone":                 status.Timezone,
		"closeTime":                status.CloseTime,
		"lastPoll":                 status.LastPoll,
		"lastSuccessfulExpiry":     status.LastSuccessfulExpiry,
		"lastError":                status.LastError,
		"dueOrders":                status.DueOrders,
		"oldestDueAgeSeconds":      status.OldestDueAge.Seconds(),
		"totalProcessedInLastPoll": status.TotalProcessedInLastPoll,
		"consecutiveErrors":        status.ConsecutiveErrors,
	})
}
