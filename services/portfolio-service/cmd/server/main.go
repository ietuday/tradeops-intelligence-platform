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

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/config"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/db"
	httpapi "github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/http"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/service"
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
	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.PortfolioTopic, cfg.SnapshotTopic)
	defer producer.Close()
	portfolioService := service.NewPortfolioService(repository.NewPortfolioRepository(pool), producer, metrics, cfg.InitialCash)

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.OrderFilledTopic, portfolioService, logger, metrics, kafka.RetryConfig{
		MaxRetries:        cfg.EventProcessingMaxRetries,
		Backoff:           time.Duration(cfg.EventProcessingBackoffMS) * time.Millisecond,
		BackoffMultiplier: cfg.EventProcessingMultiplier,
	})
	consumer.Start(ctx)
	defer consumer.Close()

	router := httpapi.NewRouter(httpapi.Dependencies{
		DB:           pool,
		KafkaBrokers: cfg.KafkaBrokers,
		Metrics:      metrics,
		Service:      portfolioService,
		Validator:    security.NewValidator([]byte(cfg.JWTSecret)),
	})
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("portfolio service started", "port", cfg.Port)
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
}
