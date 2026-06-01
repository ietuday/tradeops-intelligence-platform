package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireOne,
			BatchTimeout:           10 * time.Millisecond,
		},
	}
}

func (p *Producer) PublishTick(ctx context.Context, event domain.NormalizedTickEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Symbol),
		Value: payload,
		Time:  event.ReceivedAt,
		Headers: []kafka.Header{
			{Key: "eventType", Value: []byte(event.EventType)},
			{Key: "correlationId", Value: []byte(event.CorrelationID)},
		},
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
