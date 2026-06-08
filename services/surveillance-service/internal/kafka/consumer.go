package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Processor interface {
	ProcessEvent(context.Context, domain.SourceEvent) error
}

type Consumer struct {
	readers []readerCloser
	service Processor
	logger  *slog.Logger
}

type readerCloser interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
	Close() error
}

func NewConsumer(brokers []string, topics []string, svc Processor, logger *slog.Logger) *Consumer {
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
	return &Consumer{readers: readers, service: svc, logger: logger}
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
		}
		if err := c.service.ProcessEvent(ctx, event); err != nil {
			c.logger.Warn("failed to process surveillance source event", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset, "error", err)
		}
		if err := reader.CommitMessages(ctx, message); err != nil && ctx.Err() == nil {
			c.logger.Warn("failed to commit surveillance source event", "error", err)
		}
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
