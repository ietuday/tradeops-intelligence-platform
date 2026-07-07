package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/consumerobs"
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
	serviceName string
	topic       string
	groupID     string
	dlqTopic    string
	obsCfg      consumerobs.Config
	mu          sync.RWMutex
	status      Status
	running     bool
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

type Status struct {
	Running           bool       `json:"running"`
	LastConsumedAt    *time.Time `json:"lastConsumedAt,omitempty"`
	LastProcessedAt   *time.Time `json:"lastProcessedAt,omitempty"`
	LastDuplicateAt   *time.Time `json:"lastDuplicateAt,omitempty"`
	LastDLQAt         *time.Time `json:"lastDlqAt,omitempty"`
	ProcessedCount    int64      `json:"processedCount"`
	DuplicateCount    int64      `json:"duplicateCount"`
	DLQCount          int64      `json:"dlqCount"`
	ConsecutiveErrors int        `json:"consecutiveErrors"`
	LastError         string     `json:"lastError,omitempty"`
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
	return NewConsumerWithOptions(brokers, topic, "portfolio-service", "portfolio.dlq", svc, logger, metrics, retryConfig)
}

func NewConsumerWithOptions(brokers []string, topic, groupID, dlqTopic string, svc Processor, logger *slog.Logger, metrics *observability.Metrics, retryConfig RetryConfig) *Consumer {
	return NewConsumerWithObservability(brokers, topic, groupID, dlqTopic, svc, logger, metrics, retryConfig, consumerobs.Config{Enabled: true, LagWarnThreshold: 1000, LagCriticalThreshold: 10000, StalledAfter: 2 * time.Minute, DLQEnabled: true, DLQTopic: dlqTopic, DLQOldestAgeWarn: 5 * time.Minute, DLQOldestAgeCritical: 30 * time.Minute})
}

func NewConsumerWithObservability(brokers []string, topic, groupID, dlqTopic string, svc Processor, logger *slog.Logger, metrics *observability.Metrics, retryConfig RetryConfig, obsCfg consumerobs.Config) *Consumer {
	if obsCfg.DLQTopic == "" {
		obsCfg.DLQTopic = dlqTopic
	}
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: 0,
			StartOffset:    kafka.LastOffset,
		}),
		service:     svc,
		logger:      logger,
		metrics:     metrics,
		dlqWriter:   newDLQWriter(brokers, dlqTopic),
		retryConfig: normalizeRetryConfig(retryConfig),
		serviceName: "portfolio-service",
		topic:       topic,
		groupID:     groupID,
		dlqTopic:    dlqTopic,
		obsCfg:      obsCfg,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	go func() {
		c.setRunning(true)
		defer c.setRunning(false)
		for {
			message, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Warn("failed to fetch portfolio source event", "error", err)
				continue
			}
			c.recordConsumed()
			if err := c.processWithRetry(ctx, message); err != nil {
				c.recordError(err)
				c.logger.Warn("failed to process portfolio source event", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset, "error", err)
			} else {
				c.recordProcessed()
			}
			if err := c.reader.CommitMessages(ctx, message); err != nil && ctx.Err() == nil {
				c.logger.Warn("failed to commit portfolio source event", "error", err)
			}
		}
	}()
}

func (c *Consumer) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status := c.status
	status.Running = c.running
	return status
}

func (c *Consumer) ConsumerObsStatus() consumerobs.Snapshot {
	c.mu.RLock()
	status := c.status
	running := c.running
	c.mu.RUnlock()
	now := time.Now().UTC()
	consumerStatus := consumerobs.ConsumerStatus{
		ConsumerGroup:     c.groupID,
		Topic:             c.topic,
		Partitions:        []consumerobs.PartitionStatus{{Partition: 0, CurrentOffset: 0, LatestOffset: 0, LagMessages: 0}},
		TotalLagMessages:  0,
		LastProcessedAt:   status.LastProcessedAt,
		LastMessageAt:     status.LastConsumedAt,
		ConsecutiveErrors: status.ConsecutiveErrors,
		Status:            consumerobs.ClassifyConsumer(0, status.ConsecutiveErrors, status.LastProcessedAt, now, c.obsCfg),
	}
	if !running && c.obsCfg.Enabled {
		consumerStatus.Status = consumerobs.StatusUnknown
	}
	if c.metrics != nil && status.LastProcessedAt != nil {
		c.metrics.ConsumerObs.ConsumerLastProcessed.WithLabelValues(c.serviceName, c.groupID, c.topic).Set(float64(status.LastProcessedAt.Unix()))
	}
	dlqOldestAge := time.Duration(0)
	if status.LastDLQAt != nil {
		dlqOldestAge = now.Sub(status.LastDLQAt.UTC())
	}
	if c.metrics != nil {
		c.metrics.ConsumerObs.ConsumerLagMessages.WithLabelValues(c.serviceName, c.groupID, c.topic, "0").Set(0)
		c.metrics.ConsumerObs.ConsumerLagOldestAge.WithLabelValues(c.serviceName, c.groupID, c.topic).Set(0)
		c.metrics.ConsumerObs.DLQMessages.WithLabelValues(c.serviceName, c.dlqTopic).Set(float64(status.DLQCount))
		c.metrics.ConsumerObs.DLQOldestMessageAge.WithLabelValues(c.serviceName, c.dlqTopic).Set(dlqOldestAge.Seconds())
	}
	dlq := consumerobs.DLQStatus{
		Topic:                   c.dlqTopic,
		MessageCount:            status.DLQCount,
		OldestMessageAgeSeconds: dlqOldestAge.Seconds(),
		NewestMessageAgeSeconds: dlqOldestAge.Seconds(),
		LastObservedAt:          status.LastDLQAt,
		Status:                  consumerobs.ClassifyDLQ(status.DLQCount, dlqOldestAge, c.obsCfg),
	}
	return consumerobs.Snapshot{
		Service:   c.serviceName,
		Status:    consumerobs.AggregateStatus([]consumerobs.ConsumerStatus{consumerStatus}, []consumerobs.DLQStatus{dlq}),
		CheckedAt: now,
		Consumers: []consumerobs.ConsumerStatus{consumerStatus},
		DLQ:       []consumerobs.DLQStatus{dlq},
		Enabled:   c.obsCfg.Enabled,
	}
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
			c.metrics.ConsumerObs.ConsumerProcessingTotal.WithLabelValues(c.serviceName, c.groupID, message.Topic, eventTypeFromMessage(message.Value), status).Inc()
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
			c.metrics.ConsumerObs.ConsumerProcessingErrors.WithLabelValues(c.serviceName, c.groupID, message.Topic, eventTypeFromMessage(message.Value), "processing_error").Inc()
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
		c.metrics.ConsumerObs.DLQEvents.WithLabelValues(c.serviceName, c.dlqTopic, eventTypeFromMessage(message.Value), "processing_error").Inc()
		c.metrics.ConsumerObs.DLQMessages.WithLabelValues(c.serviceName, c.dlqTopic).Inc()
	}
	c.recordDLQ()
	c.logger.Warn("published portfolio event to DLQ", "topic", message.Topic, "error", lastErr)
	return lastErr
}

func (c *Consumer) recordConsumed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := time.Now().UTC()
	c.status.LastConsumedAt = &t
}

func (c *Consumer) recordProcessed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := time.Now().UTC()
	c.status.LastProcessedAt = &t
	c.status.ProcessedCount++
	c.status.ConsecutiveErrors = 0
	c.status.LastError = ""
}

func (c *Consumer) recordError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.ConsecutiveErrors++
	c.status.LastError = err.Error()
}

func (c *Consumer) recordDLQ() {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := time.Now().UTC()
	c.status.LastDLQAt = &t
	c.status.DLQCount++
}

func (c *Consumer) setRunning(running bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = running
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

func eventTypeFromMessage(payload []byte) string {
	var body struct {
		EventType string `json:"eventType"`
	}
	if err := json.Unmarshal(payload, &body); err != nil || body.EventType == "" {
		return "unknown"
	}
	return body.EventType
}
