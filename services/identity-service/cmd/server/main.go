package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/config"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/db"
	httpapi "github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/http"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.ConnectPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.ApplyMigrations(ctx, pool, "/app/migrations"); err != nil {
		if err := db.ApplyMigrations(ctx, pool, "migrations"); err != nil {
			logger.Error("database migration failed", "error", err)
			os.Exit(1)
		}
	}

	redisClient := db.ConnectRedis(cfg.RedisAddr)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("redis connection failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	metrics := observability.NewMetrics()
	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewRefreshTokenRepository(pool, redisClient, []byte(cfg.RefreshTokenSecret))
	auditRepo := repository.NewAuditRepository(pool)
	tokenManager := security.NewTokenManager([]byte(cfg.JWTSecret), cfg.AccessTokenTTL)
	authService := service.NewAuthService(userRepo, tokenRepo, auditRepo, tokenManager, cfg.RefreshTokenTTL)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Config:       cfg,
		DB:           pool,
		Redis:        redisClient,
		Logger:       logger,
		Metrics:      metrics,
		AuthService:  authService,
		TokenManager: tokenManager,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("identity service started", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
