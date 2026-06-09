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

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/config"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/db"
	httpapi "github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	shutdownTracing, err := observability.SetupTracing(ctx, "order-service")
	if err != nil {
		logger.Warn("opentelemetry tracing disabled", "error", err)
	}

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

	metrics := observability.NewMetrics()
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	orderService := service.NewOrderService(repository.NewOrderRepository(pool), producer, metrics)
	router := httpapi.NewRouter(httpapi.Dependencies{
		DB:           pool,
		KafkaBrokers: cfg.KafkaBrokers,
		Metrics:      metrics,
		Service:      orderService,
		Validator:    security.NewValidator([]byte(cfg.JWTSecret)),
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("order service started", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	if err := shutdownTracing(shutdownCtx); err != nil {
		logger.Warn("opentelemetry shutdown failed", "error", err)
	}
}
