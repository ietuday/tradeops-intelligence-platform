package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	KafkaBrokers []string
	Metrics      *observability.Metrics
	Service      *service.SurveillanceService
	Validator    *security.Validator
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)

	health := handlers.NewHealthHandler(deps.DB, deps.KafkaBrokers)
	alerts := handlers.NewAlertHandler(deps.Service)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Route("/api/v1/surveillance", func(r chi.Router) {
		r.Use(middleware.Auth(deps.Validator))
		r.Get("/alerts", alerts.List)
		r.Get("/alerts/summary", alerts.Summary)
		r.Get("/alerts/{id}", alerts.Get)
		r.Post("/alerts/{id}/acknowledge", alerts.Acknowledge)
		r.Post("/alerts/{id}/resolve", alerts.Resolve)
		r.Post("/alerts/{id}/dismiss", alerts.Dismiss)
	})

	return router
}
