package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type Processor interface {
	ProcessOrderFilled(context.Context, []byte) error
}

type Consumer struct {
	reader  *kafka.Reader
	service Processor
	logger  *slog.Logger
}

func NewConsumer(brokers []string, topic string, svc Processor, logger *slog.Logger) *Consumer {
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
		service: svc,
		logger:  logger,
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
			if err := c.service.ProcessOrderFilled(ctx, message.Value); err != nil {
				c.logger.Warn("failed to process order filled event", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset, "error", err)
			}
			if err := c.reader.CommitMessages(ctx, message); err != nil && ctx.Err() == nil {
				c.logger.Warn("failed to commit order filled event", "error", err)
			}
		}
	}()
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
