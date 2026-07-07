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
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/expiry"
	httpapi "github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/http"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/outbox"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/risk"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/security"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/service"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/stoptrigger"
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

	calendar, err := expiry.NewWeekdayTradingCalendar(cfg.Expiry.DayTimezone, cfg.Expiry.DayCloseTime)
	if err != nil {
		logger.Error("order expiry calendar initialization failed", "error", err)
		os.Exit(1)
	}

	orderRepo := repository.NewOrderRepository(pool)
	referencePriceRepo := repository.NewReferencePriceRepository(pool)
	orderService := service.NewOrderService(orderRepo, producer, metrics, calendar, service.RiskOptions{Enabled: cfg.PreTradeRisk.Enabled, FailOpen: cfg.PreTradeRisk.FailOpen})
	if cfg.PreTradeRisk.Enabled {
		riskClient, err := risk.NewClient(risk.ClientConfig{
			BaseURL:    cfg.PreTradeRisk.URL,
			Timeout:    cfg.PreTradeRisk.Timeout,
			MaxRetries: cfg.PreTradeRisk.MaxRetries,
			BaseDelay:  cfg.PreTradeRisk.RetryBaseDelay,
		})
		if err != nil {
			logger.Error("pre-trade risk client initialization failed", "error", err)
			os.Exit(1)
		}
		orderService.SetRiskChecker(riskClient)
		logger.Info("pre-trade risk evaluation enabled", "failOpen", cfg.PreTradeRisk.FailOpen)
	} else {
		logger.Warn("pre-trade risk evaluation disabled")
	}
	var expiryDone chan struct{}
	var expiryWorker *expiry.Worker
	if cfg.Expiry.Enabled {
		workerID, _ := os.Hostname()
		if workerID == "" {
			workerID = "order-service"
		}
		expiryCfg := expiry.Config{
			Enabled:           cfg.Expiry.Enabled,
			PollInterval:      cfg.Expiry.PollInterval,
			BatchSize:         cfg.Expiry.BatchSize,
			MaxBatchesPerPoll: cfg.Expiry.MaxBatchesPerPoll,
			ProcessingTimeout: cfg.Expiry.ProcessingTimeout,
			ShutdownTimeout:   cfg.Expiry.ShutdownTimeout,
			DayTimezone:       cfg.Expiry.DayTimezone,
			DayCloseTime:      cfg.Expiry.DayCloseTime,
		}
		expiryWorker = expiry.NewWorker(expiry.NewRepository(pool, workerID), metrics, logger, expiryCfg)
		expiryDone = make(chan struct{})
		go func() {
			defer close(expiryDone)
			if err := expiryWorker.Run(ctx); err != nil {
				logger.Error("order expiry worker failed", "error", err)
			}
		}()
	} else {
		logger.Info("order expiry worker disabled")
	}
	var stopTriggerDone chan struct{}
	var stopTriggerWorker *stoptrigger.Worker
	var priceConsumerDone chan struct{}
	var priceConsumer *stoptrigger.PriceConsumer
	if cfg.StopTrigger.Enabled {
		stopTriggerCfg := stoptrigger.Config{
			Enabled:              cfg.StopTrigger.Enabled,
			PollInterval:         cfg.StopTrigger.PollInterval,
			BatchSize:            cfg.StopTrigger.BatchSize,
			MaxReferencePriceAge: cfg.StopTrigger.MaxReferencePriceAge,
			ProcessingTimeout:    cfg.StopTrigger.ProcessingTimeout,
			ShutdownTimeout:      cfg.StopTrigger.ShutdownTimeout,
			MarketTopic:          cfg.StopTrigger.MarketTopic,
			ConsumerGroup:        cfg.StopTrigger.ConsumerGroup,
		}
		stopTriggerWorker = stoptrigger.NewWorker(orderRepo, metrics, logger, stopTriggerCfg)
		stopTriggerDone = make(chan struct{})
		go func() {
			defer close(stopTriggerDone)
			if err := stopTriggerWorker.Run(ctx); err != nil {
				logger.Error("stop trigger worker failed", "error", err)
			}
		}()
		priceConsumer = stoptrigger.NewPriceConsumer(cfg.KafkaBrokers, cfg.StopTrigger.MarketTopic, cfg.StopTrigger.ConsumerGroup, referencePriceRepo, metrics, logger)
		priceConsumerDone = make(chan struct{})
		go func() {
			defer close(priceConsumerDone)
			if err := priceConsumer.Run(ctx); err != nil {
				logger.Error("market price consumer failed", "error", err)
			}
		}()
		logger.Info("stop trigger worker enabled", "pollInterval", cfg.StopTrigger.PollInterval, "batchSize", cfg.StopTrigger.BatchSize, "marketTopic", cfg.StopTrigger.MarketTopic)
	} else {
		logger.Info("stop trigger worker disabled")
	}
	var outboxDone chan struct{}
	var outboxPublisher *outbox.Publisher
	if cfg.Outbox.Enabled {
		publisher, err := outbox.NewPublisher(outbox.NewRepository(pool), producer, metrics, logger, cfg.Outbox)
		if err != nil {
			logger.Error("outbox publisher initialization failed", "error", err)
			os.Exit(1)
		}
		outboxPublisher = publisher
		outboxDone = make(chan struct{})
		go func() {
			defer close(outboxDone)
			if err := publisher.Run(ctx); err != nil {
				logger.Error("outbox publisher failed", "error", err)
			}
		}()
	} else {
		logger.Info("outbox publishing disabled")
	}
	router := httpapi.NewRouter(httpapi.Dependencies{
		DB:           pool,
		KafkaBrokers: cfg.KafkaBrokers,
		Metrics:      metrics,
		Expiry:       expiryWorker,
		StopTrigger:  stopTriggerWorker,
		Outbox:       outboxPublisher,
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	if outboxDone != nil {
		outboxCtx, outboxCancel := context.WithTimeout(context.Background(), cfg.Outbox.ShutdownTimeout)
		defer outboxCancel()
		select {
		case <-outboxDone:
		case <-outboxCtx.Done():
			logger.Warn("outbox publisher shutdown timed out")
		}
	}
	if priceConsumer != nil {
		_ = priceConsumer.Close()
	}
	if stopTriggerDone != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), cfg.StopTrigger.ShutdownTimeout)
		defer stopCancel()
		select {
		case <-stopTriggerDone:
		case <-stopCtx.Done():
			logger.Warn("stop trigger worker shutdown timed out")
		}
	}
	if priceConsumerDone != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), cfg.StopTrigger.ShutdownTimeout)
		defer stopCancel()
		select {
		case <-priceConsumerDone:
		case <-stopCtx.Done():
			logger.Warn("market price consumer shutdown timed out")
		}
	}
	if expiryDone != nil {
		expiryCtx, expiryCancel := context.WithTimeout(context.Background(), cfg.Expiry.ShutdownTimeout)
		defer expiryCancel()
		select {
		case <-expiryDone:
		case <-expiryCtx.Done():
			logger.Warn("order expiry worker shutdown timed out")
		}
	}
	if err := shutdownTracing(shutdownCtx); err != nil {
		logger.Warn("opentelemetry shutdown failed", "error", err)
	}
}
