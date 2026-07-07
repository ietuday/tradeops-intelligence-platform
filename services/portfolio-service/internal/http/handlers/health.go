package handlers

import (
	"context"
	"net/http"
	"time"

	httpmiddleware "github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/http/middleware"
	portfoliokafka "github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/outbox"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type HealthHandler struct {
	db           *pgxpool.Pool
	kafkaBrokers []string
	outbox       interface{ Status() outbox.Status }
	consumer     interface{ Status() portfoliokafka.Status }
}

func NewHealthHandler(db *pgxpool.Pool, kafkaBrokers []string, outboxProvider interface{ Status() outbox.Status }, consumerProvider interface{ Status() portfoliokafka.Status }) *HealthHandler {
	return &HealthHandler{db: db, kafkaBrokers: kafkaBrokers, outbox: outboxProvider, consumer: consumerProvider}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "portfolio-service"})
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
	httpmiddleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready", "service": "portfolio-service"})
}

func (h *HealthHandler) OutboxStatus(w http.ResponseWriter, _ *http.Request) {
	if h.outbox == nil {
		httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	status := h.outbox.Status()
	httpmiddleware.WriteJSON(w, http.StatusOK, status)
}

func (h *HealthHandler) ConsumerStatus(w http.ResponseWriter, _ *http.Request) {
	if h.consumer == nil {
		httpmiddleware.WriteJSON(w, http.StatusOK, map[string]any{"running": false})
		return
	}
	httpmiddleware.WriteJSON(w, http.StatusOK, h.consumer.Status())
}
