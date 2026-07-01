package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/config"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Publisher struct {
	repo     Store
	producer EventProducer
	metrics  *observability.Metrics
	logger   *slog.Logger
	cfg      config.OutboxConfig
	backoff  Backoff
	workerID string

	mu     sync.RWMutex
	status Stats
}

type Store interface {
	Claim(ctx context.Context, workerID string, batchSize, maxAttempts int, leaseDuration time.Duration) ([]Record, int64, error)
	MarkPublished(ctx context.Context, id, workerID string) (bool, error)
	MarkFailed(ctx context.Context, id, workerID string, nextAvailableAt time.Time, safeError string) (bool, error)
	MarkTerminal(ctx context.Context, id, workerID, safeError string) (bool, error)
	Snapshot(ctx context.Context) (Stats, error)
}

func NewPublisher(repo Store, producer EventProducer, metrics *observability.Metrics, logger *slog.Logger, cfg config.OutboxConfig) (*Publisher, error) {
	if repo == nil {
		return nil, errors.New("outbox repository is required")
	}
	if producer == nil {
		return nil, errors.New("outbox producer is required")
	}
	if metrics == nil {
		return nil, errors.New("outbox metrics are required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	backoff, err := NewBackoff(cfg.BaseBackoff, cfg.MaxBackoff)
	if err != nil {
		return nil, err
	}
	return &Publisher{
		repo:     repo,
		producer: producer,
		metrics:  metrics,
		logger:   logger,
		cfg:      cfg,
		backoff:  backoff,
		workerID: workerID(),
	}, nil
}

func (p *Publisher) Run(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.logger.Info("outbox publisher disabled")
		return nil
	}
	p.logger.Info("outbox publisher started", "worker_id", p.workerID, "config", p.cfg.String())
	defer p.logger.Info("outbox publisher stopped", "worker_id", p.workerID)

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()
	for {
		p.poll(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (p *Publisher) Status() Stats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *Publisher) poll(ctx context.Context) {
	ctx, span := otel.Tracer("order-service").Start(ctx, "outbox.poll")
	defer span.End()
	now := time.Now().UTC()
	p.setLastPoll(now)

	start := time.Now()
	records, recovered, err := p.repo.Claim(ctx, p.workerID, p.cfg.BatchSize, p.cfg.MaxAttempts, p.cfg.LeaseDuration)
	p.metrics.OutboxClaimDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		p.recordError("claim failed", err)
		span.RecordError(err)
		return
	}
	if recovered > 0 {
		p.metrics.OutboxLeaseRecoveries.Add(float64(recovered))
	}
	p.metrics.OutboxClaimed.Add(float64(len(records)))
	p.metrics.OutboxBatchSize.Observe(float64(len(records)))
	if len(records) == 0 {
		p.refreshSnapshot(ctx)
		return
	}
	for _, record := range records {
		if ctx.Err() != nil {
			break
		}
		p.publishOne(ctx, record)
	}
	p.refreshSnapshot(ctx)
}

func (p *Publisher) publishOne(ctx context.Context, record Record) {
	ctx, span := otel.Tracer("order-service").Start(ctx, "outbox.publish")
	defer span.End()
	span.SetAttributes(
		attribute.String("messaging.destination", record.Topic),
		attribute.String("outbox.event_type", record.EventType),
		attribute.String("outbox.aggregate_type", record.AggregateType),
		attribute.Int("outbox.attempt", record.AttemptCount+1),
	)

	if !json.Valid(record.Payload) {
		p.terminal(ctx, record, "invalid JSON payload")
		return
	}

	headers := headers(record)
	publishCtx := observability.ContextWithTraceParent(context.Background(), record.TraceParent)
	publishCtx, cancel := context.WithTimeout(publishCtx, p.cfg.PublishTimeout)
	defer cancel()

	start := time.Now()
	err := p.producer.PublishRaw(publishCtx, record.Topic, []byte(messageKey(record)), record.Payload, headers)
	duration := time.Since(start)
	p.metrics.OutboxPublishDuration.WithLabelValues(record.EventType, record.Topic, resultLabel(err)).Observe(duration.Seconds())
	if err != nil {
		p.retryOrTerminal(ctx, record, err)
		return
	}

	claimCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	claimCtx, markSpan := otel.Tracer("order-service").Start(claimCtx, "outbox.mark_published")
	updated, err := p.repo.MarkPublished(claimCtx, record.ID, p.workerID)
	markSpan.End()
	if err != nil {
		p.recordError("mark published failed", err)
		return
	}
	if !updated {
		p.logger.Warn("outbox ownership lost before mark published", "event_type", record.EventType, "topic", record.Topic, "worker_id", p.workerID, "correlation_id", record.CorrelationID)
		return
	}
	p.metrics.OutboxPublished.WithLabelValues(record.EventType, record.Topic).Inc()
	p.setLastPublished(time.Now().UTC())
	p.clearConsecutiveErrors()
}

func (p *Publisher) retryOrTerminal(ctx context.Context, record Record, err error) {
	safe := sanitizeError(err, p.cfg.ErrorMaxLength)
	if record.AttemptCount+1 >= p.cfg.MaxAttempts {
		p.terminal(ctx, record, safe)
		return
	}
	delay := p.backoff.Delay(record.AttemptCount)
	p.metrics.OutboxRetryDelay.Observe(delay.Seconds())
	next := time.Now().UTC().Add(delay)
	markCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	markCtx, span := otel.Tracer("order-service").Start(markCtx, "outbox.mark_failed")
	updated, markErr := p.repo.MarkFailed(markCtx, record.ID, p.workerID, next, safe)
	span.End()
	if markErr != nil {
		p.recordError("mark retry failed", markErr)
		return
	}
	if !updated {
		p.logger.Warn("outbox ownership lost before mark failed", "event_type", record.EventType, "topic", record.Topic, "worker_id", p.workerID, "correlation_id", record.CorrelationID)
		return
	}
	p.metrics.OutboxPublishErrors.WithLabelValues(record.EventType, record.Topic, "retry").Inc()
	p.incrementConsecutiveErrors(safe)
	p.logger.Warn("outbox publish failed; scheduled retry", "event_type", record.EventType, "topic", record.Topic, "attempt", record.AttemptCount+1, "retry_delay", delay.String(), "worker_id", p.workerID, "correlation_id", record.CorrelationID, "error", safe)
}

func (p *Publisher) terminal(ctx context.Context, record Record, reason string) {
	markCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	markCtx, span := otel.Tracer("order-service").Start(markCtx, "outbox.mark_failed")
	updated, err := p.repo.MarkTerminal(markCtx, record.ID, p.workerID, sanitizeText(reason, p.cfg.ErrorMaxLength))
	span.End()
	if err != nil {
		p.recordError("mark terminal failed", err)
		return
	}
	if !updated {
		p.logger.Warn("outbox ownership lost before terminal failure", "event_type", record.EventType, "topic", record.Topic, "worker_id", p.workerID, "correlation_id", record.CorrelationID)
		return
	}
	p.metrics.OutboxPublishErrors.WithLabelValues(record.EventType, record.Topic, "terminal").Inc()
	p.metrics.OutboxTerminalFailures.WithLabelValues(record.EventType, record.Topic).Inc()
	p.incrementConsecutiveErrors(reason)
	p.logger.Error("outbox event terminally failed", "event_type", record.EventType, "topic", record.Topic, "attempt", record.AttemptCount+1, "worker_id", p.workerID, "correlation_id", record.CorrelationID, "error", sanitizeText(reason, p.cfg.ErrorMaxLength))
}

func (p *Publisher) refreshSnapshot(ctx context.Context) {
	stats, err := p.repo.Snapshot(ctx)
	if err != nil {
		p.recordError("outbox snapshot failed", err)
		return
	}
	p.metrics.OutboxPending.Set(float64(stats.Pending))
	p.metrics.OutboxProcessing.Set(float64(stats.Processing))
	p.metrics.OutboxFailed.Set(float64(stats.Failed))
	p.metrics.OutboxOldestPendingAge.Set(stats.OldestPendingAge.Seconds())

	p.mu.Lock()
	stats.LastPoll = p.status.LastPoll
	stats.LastPublished = p.status.LastPublished
	stats.LastError = p.status.LastError
	stats.ConsecutiveErrors = p.status.ConsecutiveErrors
	p.status = stats
	p.mu.Unlock()
}

func (p *Publisher) setLastPoll(value time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.LastPoll = &value
}

func (p *Publisher) setLastPublished(value time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.LastPublished = &value
}

func (p *Publisher) clearConsecutiveErrors() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.ConsecutiveErrors = 0
	p.status.LastError = ""
}

func (p *Publisher) incrementConsecutiveErrors(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.ConsecutiveErrors++
	p.status.LastError = sanitizeText(message, p.cfg.ErrorMaxLength)
}

func (p *Publisher) recordError(message string, err error) {
	safe := sanitizeError(err, p.cfg.ErrorMaxLength)
	p.incrementConsecutiveErrors(safe)
	p.logger.Error(message, "error", safe, "worker_id", p.workerID)
}

func headers(record Record) map[string]string {
	h := map[string]string{
		"event-id":     record.EventID,
		"event-type":   record.EventType,
		"tenant-id":    record.TenantID,
		"content-type": "application/json",
	}
	if record.CorrelationID != "" {
		h["correlation-id"] = record.CorrelationID
	}
	if record.TraceParent != "" {
		h["traceparent"] = record.TraceParent
	}
	if record.TraceState != "" {
		h["tracestate"] = record.TraceState
	}
	return h
}

func messageKey(record Record) string {
	return record.TenantID + ":" + record.AggregateID
}

func resultLabel(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}

func sanitizeError(err error, max int) string {
	if err == nil {
		return ""
	}
	return sanitizeText(err.Error(), max)
}

func sanitizeText(value string, max int) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	if max <= 0 {
		max = 512
	}
	if len(value) > max {
		return value[:max]
	}
	return value
}

func workerID() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%d-%s", host, os.Getpid(), uuid.NewString())
}
