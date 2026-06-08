package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/observability"
	"github.com/segmentio/kafka-go"
)

type Processor interface {
	ProcessOrderFilled(context.Context, []byte) error
}

type Consumer struct {
	reader      readerCloser
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

func NewConsumer(brokers []string, topic string, svc Processor, logger *slog.Logger, metrics *observability.Metrics, retryConfig RetryConfig) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        "portfolio-service",
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.LastOffset,
		}),
		service:     svc,
		logger:      logger,
		metrics:     metrics,
		dlqWriter:   newDLQWriter(brokers, "portfolio.dlq"),
		retryConfig: normalizeRetryConfig(retryConfig),
	}
}

func (c *Consumer) Start(ctx context.Context) {
	go func() {
		for {
			message, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Warn("failed to fetch order filled event", "error", err)
				continue
			}
			if err := c.processWithRetry(ctx, message); err != nil {
				c.logger.Warn("failed to process order filled event", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset, "error", err)
			}
			if err := c.reader.CommitMessages(ctx, message); err != nil && ctx.Err() == nil {
				c.logger.Warn("failed to commit order filled event", "error", err)
			}
		}
	}()
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		_ = c.dlqWriter.Close()
		return err
	}
	return c.dlqWriter.Close()
}

func (c *Consumer) processWithRetry(ctx context.Context, message kafka.Message) error {
	var lastErr error
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		err := c.service.ProcessOrderFilled(ctx, message.Value)
		status := "success"
		if err != nil {
			status = "failed"
		}
		if c.metrics != nil {
			c.metrics.ProcessingAttempts.WithLabelValues(message.Topic, status).Inc()
		}
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt >= c.retryConfig.MaxRetries {
			break
		}
		if c.metrics != nil {
			c.metrics.EventsRetried.WithLabelValues(message.Topic).Inc()
		}
		if err := sleepWithContext(ctx, retryDelay(c.retryConfig, attempt)); err != nil {
			return err
		}
	}
	if err := c.publishDLQ(ctx, message, lastErr, c.retryConfig.MaxRetries); err != nil {
		c.logger.Warn("failed to publish portfolio DLQ event", "topic", message.Topic, "error", err)
		return lastErr
	}
	if c.metrics != nil {
		c.metrics.EventsDeadlettered.WithLabelValues(message.Topic).Inc()
	}
	c.logger.Warn("published portfolio event to DLQ", "topic", message.Topic, "error", lastErr)
	return lastErr
}

func (c *Consumer) publishDLQ(ctx context.Context, message kafka.Message, processingErr error, retryCount int) error {
	if c.dlqWriter == nil {
		return nil
	}
	correlationID := headerValue(message.Headers, "correlationId")
	payload, err := json.Marshal(DLQEvent{
		OriginalTopic:   message.Topic,
		OriginalPayload: string(message.Value),
		ErrorMessage:    processingErr.Error(),
		ServiceName:     "portfolio-service",
		FailedAt:        time.Now().UTC(),
		CorrelationID:   correlationID,
		RetryCount:      retryCount,
	})
	if err != nil {
		return err
	}
	return c.dlqWriter.WriteMessages(ctx, kafka.Message{
		Key:   message.Key,
		Value: payload,
		Time:  time.Now().UTC(),
		Headers: []kafka.Header{
			{Key: "originalTopic", Value: []byte(message.Topic)},
			{Key: "correlationId", Value: []byte(correlationID)},
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
