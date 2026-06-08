package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/audit-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB        *pgxpool.Pool
	Metrics   *observability.Metrics
	Service   *service.AuditService
	Validator *security.Validator
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)

	health := handlers.NewHealthHandler(deps.DB)
	audit := handlers.NewAuditHandler(deps.Service)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Group(func(r chi.Router) {
		r.Use(middleware.Auth(deps.Validator))
		r.Get("/api/v1/audit/logs", audit.List)
		r.Get("/api/v1/audit/logs/{id}", audit.Get)
		r.Get("/api/v1/audit/summary", audit.Summary)
		r.Get("/api/v1/audit/export", audit.Export)
	})

	return router
}
