package kafka

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	brokers []string
	writers map[string]*kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{brokers: brokers, writers: map[string]*kafka.Writer{}}
}

func (p *Producer) Publish(ctx context.Context, event domain.OrderEvent) error {
	topic := topicForEvent(event.EventType)
	writer := p.writer(topic)
	event.TraceParent = observability.TraceParent(ctx)
	event.TraceID, event.SpanID = observability.TraceIDs(ctx)
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	headers := []kafka.Header{
		{Key: "eventType", Value: []byte(event.EventType)},
		{Key: "correlationId", Value: []byte(event.CorrelationID)},
	}
	if event.TraceParent != "" {
		headers = append(headers, kafka.Header{Key: "traceparent", Value: []byte(event.TraceParent)})
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.OrderID),
		Value:   payload,
		Time:    event.OccurredAt,
		Headers: headers,
	})
}

func (p *Producer) Close() error {
	var first error
	for _, writer := range p.writers {
		if err := writer.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (p *Producer) writer(topic string) *kafka.Writer {
	if writer, ok := p.writers[topic]; ok {
		return writer
	}
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(p.brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           10 * time.Millisecond,
	}
	p.writers[topic] = writer
	return writer
}

func topicForEvent(eventType string) string {
	return strings.ReplaceAll(eventType, "order.", "order.")
}
