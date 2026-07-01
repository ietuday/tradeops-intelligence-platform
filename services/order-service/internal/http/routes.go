package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/expiry"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/outbox"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	KafkaBrokers []string
	Metrics      *observability.Metrics
	Expiry       interface{ Status() expiry.Stats }
	Outbox       interface{ Status() outbox.Stats }
	Service      *service.OrderService
	Validator    *security.Validator
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)
	router.Use(observability.TraceAttributes("order-service"))

	health := handlers.NewHealthHandler(deps.DB, deps.KafkaBrokers, deps.Outbox, deps.Expiry)
	orders := handlers.NewOrderHandler(deps.Service)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Get("/internal/outbox/status", health.OutboxStatus)
	router.Get("/internal/order-expiry/status", health.OrderExpiryStatus)
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Group(func(r chi.Router) {
		r.Use(middleware.Auth(deps.Validator))
		r.Post("/orders", orders.Create)
		r.Get("/orders", orders.List)
		r.Get("/orders/{id}", orders.Get)
		r.Patch("/orders/{id}", orders.Amend)
		r.Post("/orders/{id}/cancel", orders.Cancel)
		r.Get("/orders/{id}/executions", orders.Executions)
		r.Get("/order-books/{symbol}", orders.OrderBookDepth)
		r.Get("/order-books/{symbol}/depth", orders.OrderBookDepth)
		r.Get("/trades", orders.Trades)
		r.Get("/trades/{id}", orders.Trade)
	})

	return observability.HTTPHandler("order-service", router)
}
