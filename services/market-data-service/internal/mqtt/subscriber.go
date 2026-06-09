package mqtt

import (
	"context"
	"log/slog"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/service"
)

type Subscriber struct {
	client  paho.Client
	topic   string
	service *service.MarketDataService
	metrics *observability.Metrics
	logger  *slog.Logger
}

func NewSubscriber(broker, topic string, svc *service.MarketDataService, metrics *observability.Metrics, logger *slog.Logger) *Subscriber {
	options := paho.NewClientOptions().
		AddBroker(broker).
		SetClientID("tradeops-market-data-service-" + uuid.NewString()).
		SetConnectRetry(true).
		SetConnectRetryInterval(2 * time.Second).
		SetAutoReconnect(true)
	options.OnConnect = func(client paho.Client) {
		metrics.MQTTStatus.Set(1)
	}
	options.OnConnectionLost = func(_ paho.Client, err error) {
		metrics.MQTTStatus.Set(0)
		logger.Warn("mqtt connection lost", "error", err)
	}
	return &Subscriber{
		client:  paho.NewClient(options),
		topic:   topic,
		service: svc,
		metrics: metrics,
		logger:  logger,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	if token := s.client.Connect(); token.Wait() && token.Error() != nil {
		s.metrics.MQTTStatus.Set(0)
		return token.Error()
	}
	s.metrics.MQTTStatus.Set(1)

	if token := s.client.Subscribe(s.topic, 1, func(_ paho.Client, msg paho.Message) {
		correlationID := uuid.NewString()
		if err := s.service.HandleTickPayload(context.Background(), msg.Payload(), correlationID); err != nil {
			s.logger.Warn("market tick rejected or failed", "topic", msg.Topic(), "correlationId", correlationID, "error", err)
		}
	}); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	go func() {
		<-ctx.Done()
		s.client.Disconnect(250)
		s.metrics.MQTTStatus.Set(0)
	}()
	return nil
}

func (s *Subscriber) Client() paho.Client {
	return s.client
}
