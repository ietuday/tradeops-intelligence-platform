package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	KafkaBrokers []string
	Metrics      *observability.Metrics
	Service      *service.OrderService
	Validator    *security.Validator
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)
	router.Use(observability.TraceAttributes("order-service"))

	health := handlers.NewHealthHandler(deps.DB, deps.KafkaBrokers)
	orders := handlers.NewOrderHandler(deps.Service)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Group(func(r chi.Router) {
		r.Use(middleware.Auth(deps.Validator))
		r.Post("/orders", orders.Create)
		r.Get("/orders", orders.List)
		r.Get("/orders/{id}", orders.Get)
		r.Post("/orders/{id}/cancel", orders.Cancel)
	})

	return observability.HTTPHandler("order-service", router)
}
