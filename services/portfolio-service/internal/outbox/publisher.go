package outbox

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/observability"
)

type Config struct {
	Enabled      bool
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
}

type Store interface {
	FetchPending(context.Context, int) ([]Event, error)
	MarkPublished(context.Context, string, time.Time) error
	MarkFailed(context.Context, string, error, time.Time, bool) error
	Snapshot(context.Context) (Status, error)
}

type Producer interface {
	PublishRaw(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error
}

type Publisher struct {
	store   Store
	prod    Producer
	metrics *observability.Metrics
	logger  *slog.Logger
	cfg     Config

	mu      sync.RWMutex
	status  Status
	running bool
}

func NewPublisher(store Store, prod Producer, metrics *observability.Metrics, logger *slog.Logger, cfg Config) (*Publisher, error) {
	if store == nil {
		return nil, errors.New("outbox store is required")
	}
	if prod == nil {
		return nil, errors.New("outbox producer is required")
	}
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	cfg = normalizeConfig(cfg)
	return &Publisher{store: store, prod: prod, metrics: metrics, logger: logger, cfg: cfg, status: Status{Enabled: cfg.Enabled, PollInterval: cfg.PollInterval.String(), BatchSize: cfg.BatchSize}}, nil
}

func (p *Publisher) Run(ctx context.Context) error {
	if !p.cfg.Enabled {
		return nil
	}
	p.setRunning(true)
	defer p.setRunning(false)
	p.poll(ctx)
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Publisher) Status() Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	status := p.status
	status.Running = p.running
	status.Enabled = p.cfg.Enabled
	status.PollInterval = p.cfg.PollInterval.String()
	status.BatchSize = p.cfg.BatchSize
	return status
}

func (p *Publisher) poll(ctx context.Context) {
	p.refreshStatus(ctx)
	events, err := p.store.FetchPending(ctx, p.cfg.BatchSize)
	if err != nil {
		p.recordError("fetch_pending", err)
		return
	}
	for _, event := range events {
		if ctx.Err() != nil {
			return
		}
		p.publishOne(ctx, event)
	}
	p.clearError()
	p.refreshStatus(ctx)
}

func (p *Publisher) publishOne(ctx context.Context, event Event) {
	status := "published"
	err := p.prod.PublishRaw(ctx, event.Topic, []byte(event.TenantID+":"+event.AggregateID), event.Payload, event.Headers)
	if err == nil {
		if markErr := p.store.MarkPublished(ctx, event.EventID, time.Now().UTC()); markErr != nil {
			p.recordError("mark_published", markErr)
			return
		}
		p.metrics.PortfolioOutboxPublishAttempts.WithLabelValues(event.EventType, status).Inc()
		p.metrics.PortfolioOutboxEvents.WithLabelValues(StatusPublished).Inc()
		return
	}
	terminal := event.Attempts+1 >= p.cfg.MaxAttempts
	status = "retry_scheduled"
	if terminal {
		status = "failed"
	}
	p.metrics.PortfolioOutboxPublishAttempts.WithLabelValues(event.EventType, status).Inc()
	p.metrics.PortfolioOutboxPublishErrors.WithLabelValues(event.EventType, "publish_error").Inc()
	next := time.Now().UTC().Add(backoff(p.cfg, event.Attempts))
	if markErr := p.store.MarkFailed(ctx, event.EventID, err, next, terminal); markErr != nil {
		p.recordError("mark_failed", markErr)
	}
}

func (p *Publisher) refreshStatus(ctx context.Context) {
	status, err := p.store.Snapshot(ctx)
	if err != nil {
		p.recordError("snapshot", err)
		return
	}
	p.metrics.PortfolioOutboxPending.Set(float64(status.Pending))
	p.metrics.PortfolioOutboxOldestPendingAge.Set(status.OldestPendingAgeSeconds)
	p.mu.Lock()
	status.Enabled = p.cfg.Enabled
	status.Running = p.running
	status.PollInterval = p.cfg.PollInterval.String()
	status.BatchSize = p.cfg.BatchSize
	status.ConsecutiveErrors = p.status.ConsecutiveErrors
	status.LastError = p.status.LastError
	p.status = status
	p.mu.Unlock()
}

func (p *Publisher) recordError(reason string, err error) {
	p.logger.Warn("portfolio outbox publisher error", "reason", reason, "error", err)
	p.metrics.PortfolioOutboxPublishErrors.WithLabelValues("unknown", reason).Inc()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.ConsecutiveErrors++
	p.status.LastError = err.Error()
}

func (p *Publisher) clearError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.ConsecutiveErrors = 0
	p.status.LastError = ""
}

func (p *Publisher) setRunning(running bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = running
}

func normalizeConfig(cfg Config) Config {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 10
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = time.Minute
	}
	return cfg
}

func backoff(cfg Config, attempts int) time.Duration {
	delay := time.Duration(float64(cfg.BaseBackoff) * math.Pow(2, float64(attempts)))
	if delay > cfg.MaxBackoff {
		return cfg.MaxBackoff
	}
	return delay
}
