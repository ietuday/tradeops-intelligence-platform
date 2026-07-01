package expiry

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"go.opentelemetry.io/otel"
)

type Worker struct {
	repo    *Repository
	metrics *observability.Metrics
	logger  *slog.Logger
	cfg     Config

	mu      sync.RWMutex
	stats   Stats
	running bool
}

func NewWorker(repo *Repository, metrics *observability.Metrics, logger *slog.Logger, cfg Config) *Worker {
	return &Worker{
		repo:    repo,
		metrics: metrics,
		logger:  logger,
		cfg:     cfg,
		stats: Stats{
			Enabled:   cfg.Enabled,
			Timezone:  cfg.DayTimezone,
			CloseTime: cfg.DayCloseTime,
		},
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.cfg.Enabled {
		w.logger.Info("order expiry worker disabled")
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

func (w *Worker) poll(ctx context.Context) {
	start := time.Now()
	ctx, span := otel.Tracer("order-service").Start(ctx, "order_expiry.poll")
	defer span.End()

	now := time.Now().UTC()
	due, oldestAge, err := w.repo.DueStats(ctx, now)
	if err != nil {
		w.recordPollError("due_stats", err)
		return
	}
	w.metrics.OrderExpiryDueOrders.Set(float64(due))
	w.metrics.OrderExpiryOldestDueAge.Set(oldestAge.Seconds())

	processed := 0
	expired := 0
	conflicts := 0
	for batch := 0; batch < w.cfg.MaxBatchesPerPoll; batch++ {
		if ctx.Err() != nil {
			break
		}
		batchCtx, cancel := context.WithTimeout(ctx, w.cfg.ProcessingTimeout)
		result, err := w.repo.ExpireDueBatch(batchCtx, w.cfg.BatchSize, time.Now().UTC())
		cancel()
		if err != nil {
			w.recordPollError("process_batch", err)
			return
		}
		w.metrics.OrderExpiryBatchSize.Observe(float64(result.Processed))
		w.metrics.OrderExpiryClaimed.Add(float64(result.Processed))
		for timeInForce, reasons := range result.ExpiredByReason {
			for reason, count := range reasons {
				w.metrics.OrdersExpired.WithLabelValues(timeInForce, reason).Add(float64(count))
			}
		}
		for _, lag := range result.LagSeconds {
			w.metrics.OrderExpiryLag.Observe(lag)
		}
		processed += result.Processed
		expired += result.Expired
		conflicts += result.Conflicts
		if result.Processed < w.cfg.BatchSize {
			break
		}
	}
	if conflicts > 0 {
		w.metrics.OrderExpiryConflicts.Add(float64(conflicts))
	}
	if expired > 0 {
		t := time.Now().UTC()
		w.mu.Lock()
		w.stats.LastSuccessfulExpiry = &t
		w.mu.Unlock()
	}

	w.metrics.OrderExpiryPolls.WithLabelValues("success").Inc()
	w.metrics.OrderExpiryDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())
	w.mu.Lock()
	t := time.Now().UTC()
	w.stats.LastPoll = &t
	w.stats.LastError = ""
	w.stats.DueOrders = due
	w.stats.OldestDueAge = oldestAge
	w.stats.TotalProcessedInLastPoll = processed
	w.stats.ConsecutiveErrors = 0
	w.mu.Unlock()
}

func (w *Worker) Status() Stats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	stats := w.stats
	stats.Running = w.running
	return stats
}

func (w *Worker) recordPollError(result string, err error) {
	w.logger.Warn("order expiry poll failed", "result", result, "error", err)
	w.metrics.OrderExpiryPolls.WithLabelValues("error").Inc()
	w.metrics.OrderExpiryErrors.WithLabelValues(result).Inc()
	w.metrics.OrderExpiryDuration.WithLabelValues("error").Observe(0)
	w.mu.Lock()
	defer w.mu.Unlock()
	t := time.Now().UTC()
	w.stats.LastPoll = &t
	w.stats.LastError = err.Error()
	w.stats.ConsecutiveErrors++
}

func (w *Worker) setRunning(running bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = running
}
