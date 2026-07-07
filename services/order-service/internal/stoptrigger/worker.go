package stoptrigger

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"go.opentelemetry.io/otel"
)

type Worker struct {
	repo    Repository
	metrics *observability.Metrics
	logger  *slog.Logger
	cfg     Config

	mu      sync.RWMutex
	stats   Stats
	running bool
}

func NewWorker(repo Repository, metrics *observability.Metrics, logger *slog.Logger, cfg Config) *Worker {
	return &Worker{
		repo:    repo,
		metrics: metrics,
		logger:  logger,
		cfg:     cfg,
		stats: Stats{
			Enabled:              cfg.Enabled,
			PollInterval:         cfg.PollInterval.String(),
			BatchSize:            cfg.BatchSize,
			MaxReferencePriceAge: cfg.MaxReferencePriceAge.String(),
		},
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.cfg.Enabled {
		w.logger.Info("stop trigger worker disabled")
		return nil
	}
	w.setRunning(true)
	defer w.setRunning(false)

	w.poll(ctx)
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Worker) Poll(ctx context.Context) {
	w.poll(ctx)
}

func (w *Worker) poll(ctx context.Context) {
	ctx, span := otel.Tracer("order-service").Start(ctx, "stop_trigger.poll")
	defer span.End()

	start := time.Now()
	now := time.Now().UTC()
	pollCtx, cancel := context.WithTimeout(ctx, w.cfg.ProcessingTimeout)
	result, err := w.repo.TriggerDueStopOrders(pollCtx, w.cfg.BatchSize, w.cfg.MaxReferencePriceAge, now)
	cancel()
	if err != nil {
		w.recordPollError("db_error", err)
		return
	}
	for _, labels := range result.TriggeredLabels {
		w.metrics.StopTriggerOrdersTriggered.WithLabelValues(labels.Symbol, labels.Side, labels.OrderType).Inc()
	}
	for _, lag := range result.LagSeconds {
		w.metrics.StopTriggerLag.Observe(lag)
	}
	status := "success"
	if result.Processed == 0 {
		status = "skipped"
	}
	w.metrics.StopTriggerPolls.WithLabelValues(status).Inc()
	w.mu.Lock()
	t := time.Now().UTC()
	w.stats.LastPollAt = &t
	w.stats.LastSuccessfulPollAt = &t
	w.stats.TriggeredInLastPoll = result.Triggered
	w.stats.ConsecutiveErrors = 0
	w.stats.LastError = ""
	if result.Triggered > 0 {
		w.stats.LastTriggeredAt = &t
	}
	w.mu.Unlock()
	w.metrics.StopTriggerPollDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())
}

func (w *Worker) Status() Stats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	stats := w.stats
	stats.Running = w.running
	return stats
}

func (w *Worker) recordPollError(reason string, err error) {
	w.logger.Warn("stop trigger poll failed", "reason", reason, "error", err)
	w.metrics.StopTriggerPolls.WithLabelValues("error").Inc()
	w.metrics.StopTriggerErrors.WithLabelValues(reason).Inc()
	w.metrics.StopTriggerPollDuration.WithLabelValues("error").Observe(0)
	w.mu.Lock()
	defer w.mu.Unlock()
	t := time.Now().UTC()
	w.stats.LastPollAt = &t
	w.stats.LastError = err.Error()
	w.stats.ConsecutiveErrors++
}

func (w *Worker) setRunning(running bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = running
}
