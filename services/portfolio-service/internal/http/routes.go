package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	KafkaBrokers []string
	Metrics      *observability.Metrics
	Service      *service.PortfolioService
	Validator    *security.Validator
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)

	health := handlers.NewHealthHandler(deps.DB, deps.KafkaBrokers)
	portfolio := handlers.NewPortfolioHandler(deps.Service)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Group(func(r chi.Router) {
		r.Use(middleware.Auth(deps.Validator))
		r.Get("/portfolio", portfolio.Portfolio)
		r.Get("/portfolio/holdings", portfolio.Holdings)
		r.Get("/portfolio/snapshots", portfolio.Snapshots)
		r.Get("/portfolio/exposure", portfolio.Exposure)
		r.Get("/portfolio/pnl", portfolio.PnL)
	})
	return router
}
