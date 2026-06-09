package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/observability"
	"github.com/segmentio/kafka-go"
)

type Processor interface {
	ProcessEvent(context.Context, domain.SourceEvent) error
}

type Consumer struct {
	readers     []readerCloser
	service     Processor
	logger      *slog.Logger
	metrics     *observability.Metrics
	dlqWriter   writerCloser
	retryConfig RetryConfig
}

type RetryConfig struct {
	MaxRetries        int
	Backoff           time.Duration
	BackoffMultiplier float64
}

type DLQEvent struct {
	OriginalTopic   string    `json:"originalTopic"`
	OriginalPayload string    `json:"originalPayload"`
	ErrorMessage    string    `json:"errorMessage"`
	ServiceName     string    `json:"serviceName"`
	FailedAt        time.Time `json:"failedAt"`
	CorrelationID   string    `json:"correlationId,omitempty"`
	RetryCount      int       `json:"retryCount"`
}

type readerCloser interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
	Close() error
}

type writerCloser interface {
	WriteMessages(context.Context, ...kafka.Message) error
	Close() error
}

func NewConsumer(brokers []string, topics []string, svc Processor, logger *slog.Logger, metrics *observability.Metrics, retryConfig RetryConfig) *Consumer {
	readers := make([]readerCloser, 0, len(topics))
	for _, topic := range topics {
		readers = append(readers, kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        "surveillance-service",
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.LastOffset,
		}))
	}
	return &Consumer{
		readers:     readers,
		service:     svc,
		logger:      logger,
		metrics:     metrics,
		dlqWriter:   newDLQWriter(brokers, "surveillance.dlq"),
		retryConfig: normalizeRetryConfig(retryConfig),
	}
}

func (c *Consumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
}

func (c *Consumer) Close() error {
	var first error
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil && first == nil {
			first = err
		}
	}
	if c.dlqWriter != nil {
		if err := c.dlqWriter.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (c *Consumer) consume(ctx context.Context, reader readerCloser) {
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Warn("failed to fetch surveillance source event", "error", err)
			continue
		}
		event := domain.SourceEvent{
			Topic:         message.Topic,
			Key:           string(message.Key),
			Value:         message.Value,
			CorrelationID: headerValue(message.Headers, "correlationId"),
			TraceParent:   firstNonEmpty(headerValue(message.Headers, "traceparent"), payloadField(message.Value, "traceparent")),
		}
		processCtx := observability.ContextWithTraceParent(ctx, event.TraceParent)
		processCtx, span := observability.StartConsumerSpan(processCtx, "surveillance-service", event.Topic, event.CorrelationID, payloadField(event.Value, "tenantId"))
		if err := c.processWithRetry(processCtx, event); err != nil {
			c.logger.Warn("failed to process surveillance source event", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset, "error", err)
		}
		span.End()
		if err := reader.CommitMessages(ctx, message); err != nil && ctx.Err() == nil {
			c.logger.Warn("failed to commit surveillance source event", "error", err)
		}
	}
}

func payloadField(payload []byte, key string) string {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return ""
	}
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (c *Consumer) processWithRetry(ctx context.Context, event domain.SourceEvent) error {
	var lastErr error
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		start := time.Now()
		err := c.service.ProcessEvent(ctx, event)
		status := "success"
		if err != nil {
			status = "failed"
		}
		if c.metrics != nil {
			c.metrics.ProcessingAttempts.WithLabelValues(event.Topic, status).Inc()
			c.metrics.ProcessingDuration.WithLabelValues(event.Topic).Observe(time.Since(start).Seconds())
		}
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt >= c.retryConfig.MaxRetries {
			break
		}
		if c.metrics != nil {
			c.metrics.EventsRetried.WithLabelValues(event.Topic).Inc()
		}
		if err := sleepWithContext(ctx, retryDelay(c.retryConfig, attempt)); err != nil {
			return err
		}
	}
	if err := c.publishDLQ(ctx, event, lastErr, c.retryConfig.MaxRetries); err != nil {
		c.logger.Warn("failed to publish surveillance DLQ event", "topic", event.Topic, "error", err)
		return lastErr
	}
	if c.metrics != nil {
		c.metrics.EventsDeadlettered.WithLabelValues(event.Topic).Inc()
	}
	c.logger.Warn("published surveillance event to DLQ", "topic", event.Topic, "error", lastErr)
	return lastErr
}

func (c *Consumer) publishDLQ(ctx context.Context, event domain.SourceEvent, processingErr error, retryCount int) error {
	if c.dlqWriter == nil {
		return nil
	}
	payload, err := json.Marshal(DLQEvent{
		OriginalTopic:   event.Topic,
		OriginalPayload: string(event.Value),
		ErrorMessage:    processingErr.Error(),
		ServiceName:     "surveillance-service",
		FailedAt:        time.Now().UTC(),
		CorrelationID:   event.CorrelationID,
		RetryCount:      retryCount,
	})
	if err != nil {
		return err
	}
	return c.dlqWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Key),
		Value: payload,
		Time:  time.Now().UTC(),
		Headers: []kafka.Header{
			{Key: "originalTopic", Value: []byte(event.Topic)},
			{Key: "correlationId", Value: []byte(event.CorrelationID)},
		},
	})
}

func newDLQWriter(brokers []string, topic string) writerCloser {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           10 * time.Millisecond,
	}
}

func normalizeRetryConfig(cfg RetryConfig) RetryConfig {
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 500 * time.Millisecond
	}
	if cfg.BackoffMultiplier < 1 {
		cfg.BackoffMultiplier = 2
	}
	return cfg
}

func retryDelay(cfg RetryConfig, attempt int) time.Duration {
	return time.Duration(float64(cfg.Backoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt)))
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func headerValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}
