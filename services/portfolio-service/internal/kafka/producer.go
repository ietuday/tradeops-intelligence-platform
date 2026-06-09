package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	portfolioWriter *kafka.Writer
	snapshotWriter  *kafka.Writer
}

func NewProducer(brokers []string, portfolioTopic, snapshotTopic string) *Producer {
	return &Producer{
		portfolioWriter: newWriter(brokers, portfolioTopic),
		snapshotWriter:  newWriter(brokers, snapshotTopic),
	}
}

func (p *Producer) PublishPortfolioUpdated(ctx context.Context, event domain.PortfolioEvent) error {
	return writeJSON(ctx, p.portfolioWriter, event.UserID, event, event.OccurredAt)
}

func (p *Producer) PublishSnapshotCreated(ctx context.Context, snapshot domain.Snapshot, correlationID string) error {
	payload := map[string]any{
		"eventId":       correlationID,
		"eventType":     "portfolio.snapshot.created",
		"tenantId":      snapshot.TenantID,
		"snapshot":      snapshot,
		"occurredAt":    snapshot.CreatedAt,
		"correlationId": correlationID,
	}
	return writeJSON(ctx, p.snapshotWriter, snapshot.UserID, payload, snapshot.CreatedAt)
}

func (p *Producer) Close() error {
	if err := p.portfolioWriter.Close(); err != nil {
		_ = p.snapshotWriter.Close()
		return err
	}
	return p.snapshotWriter.Close()
}

func newWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           10 * time.Millisecond,
	}
}

func writeJSON(ctx context.Context, writer *kafka.Writer, key string, value any, at time.Time) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: payload, Time: at})
}
