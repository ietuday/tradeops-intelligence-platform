package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	KafkaBrokers []string
	Metrics      *observability.Metrics
	Service      *service.MarketDataService
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)

	health := handlers.NewHealthHandler(deps.DB, deps.KafkaBrokers)
	market := handlers.NewMarketHandler(deps.Service)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Handle("/metrics", deps.Metrics.Handler())
	router.Get("/ticks/latest", market.LatestTicks)
	router.Get("/symbols", market.Symbols)
	return router
}
