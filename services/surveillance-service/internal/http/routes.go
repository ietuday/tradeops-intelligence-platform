package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/service"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/simulation"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	KafkaBrokers []string
	Metrics      *observability.Metrics
	Service      *service.SurveillanceService
	RuleService  *service.RuleConfigService
	Simulation   *simulation.Service
	Validator    *security.Validator
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)
	router.Use(observability.TraceAttributes("surveillance-service"))

	health := handlers.NewHealthHandler(deps.DB, deps.KafkaBrokers)
	alerts := handlers.NewAlertHandler(deps.Service)
	rules := handlers.NewRuleHandler(deps.RuleService, deps.Simulation)

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
		r.Get("/rules", rules.List)
		r.Post("/rules/simulate", rules.SimulateBulk)
		r.Get("/rules/{ruleName}", rules.Get)
		r.Put("/rules/{ruleName}", rules.Update)
		r.Post("/rules/{ruleName}/simulate", rules.Simulate)
		r.Post("/rules/{ruleName}/enable", rules.Enable)
		r.Post("/rules/{ruleName}/disable", rules.Disable)
	})

	return observability.HTTPHandler("surveillance-service", router)
}
