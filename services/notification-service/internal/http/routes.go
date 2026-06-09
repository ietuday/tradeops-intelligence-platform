package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/notification-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB        *pgxpool.Pool
	Metrics   *observability.Metrics
	Service   *service.NotificationService
	Validator *security.Validator
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)
	router.Use(observability.TraceAttributes("notification-service"))

	health := handlers.NewHealthHandler(deps.DB)
	notifications := handlers.NewNotificationHandler(deps.Service)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Group(func(r chi.Router) {
		r.Use(middleware.Auth(deps.Validator))
		r.Get("/api/v1/notifications", notifications.List)
		r.Get("/api/v1/notifications/summary", notifications.Summary)
		r.Get("/api/v1/notifications/preferences", notifications.Preferences)
		r.Put("/api/v1/notifications/preferences", notifications.UpdatePreferences)
		r.Get("/api/v1/notifications/{id}", notifications.Get)
		r.Post("/api/v1/notifications/{id}/mark-read", notifications.MarkRead)
		r.Post("/api/v1/notifications/{id}/retry", notifications.Retry)
	})

	return observability.HTTPHandler("notification-service", router)
}
