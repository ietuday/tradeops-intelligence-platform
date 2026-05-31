package http

import (
	"log/slog"
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/config"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/http/handlers"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/http/middleware"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Config       config.Config
	DB           *pgxpool.Pool
	Redis        *redis.Client
	Logger       *slog.Logger
	Metrics      *observability.Metrics
	AuthService  *service.AuthService
	TokenManager *security.TokenManager
}

func NewRouter(deps Dependencies) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.CorrelationID)
	router.Use(deps.Metrics.Middleware)

	health := handlers.NewHealthHandler(deps.DB, deps.Redis)
	auth := handlers.NewAuthHandler(deps.AuthService)

	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Route("/auth", func(r chi.Router) {
		r.Post("/register", auth.Register)
		r.Post("/login", auth.Login)
		r.Post("/refresh", auth.Refresh)
		r.Post("/logout", auth.Logout)
		r.With(middleware.Auth(deps.TokenManager)).Get("/me", auth.Me)
	})

	router.NotFound(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		middleware.WriteError(w, nethttp.StatusNotFound, "not found")
	})

	return router
}
