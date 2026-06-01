package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/kafka"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/observability"
	"github.com/ietuday/tradeops-intelligence-platform/services/market-data-service/internal/repository"
)

type MarketDataService struct {
	repo     *repository.TickRepository
	producer *kafka.Producer
	metrics  *observability.Metrics
}

type incomingTick struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Source    string  `json:"source"`
	EventTime string  `json:"eventTime"`
}

func NewMarketDataService(repo *repository.TickRepository, producer *kafka.Producer, metrics *observability.Metrics) *MarketDataService {
	return &MarketDataService{repo: repo, producer: producer, metrics: metrics}
}

func (s *MarketDataService) HandleTickPayload(ctx context.Context, payload []byte, correlationID string) error {
	start := time.Now()
	defer s.metrics.ObserveProcessing(start)
	s.metrics.TicksReceived.Inc()

	tick, err := parseAndValidate(payload, correlationID)
	if err != nil {
		s.metrics.TicksInvalid.Inc()
		return err
	}
	s.metrics.TicksValid.Inc()

	if err := s.repo.StoreTick(ctx, tick); err != nil {
		return err
	}

	event := domain.NormalizedTickEvent{
		EventID:       uuid.NewString(),
		EventType:     "market.tick.received",
		Symbol:        tick.Symbol,
		Price:         tick.Price,
		Volume:        tick.Volume,
		Source:        tick.Source,
		EventTime:     tick.EventTime,
		ReceivedAt:    tick.ReceivedAt,
		CorrelationID: tick.CorrelationID,
	}
	if err := s.producer.PublishTick(ctx, event); err != nil {
		s.metrics.KafkaPublishErrors.Inc()
		return err
	}
	s.metrics.TicksPublished.Inc()
	return nil
}

func (s *MarketDataService) LatestTicks(ctx context.Context) ([]domain.Tick, error) {
	return s.repo.LatestTicks(ctx, 100)
}

func (s *MarketDataService) Symbols(ctx context.Context) ([]string, error) {
	return s.repo.Symbols(ctx)
}

func parseAndValidate(payload []byte, correlationID string) (domain.Tick, error) {
	var incoming incomingTick
	if err := json.Unmarshal(payload, &incoming); err != nil {
		return domain.Tick{}, err
	}
	incoming.Symbol = strings.ToUpper(strings.TrimSpace(incoming.Symbol))
	incoming.Source = strings.TrimSpace(incoming.Source)
	if incoming.Symbol == "" {
		return domain.Tick{}, errors.New("symbol is required")
	}
	if incoming.Price <= 0 {
		return domain.Tick{}, errors.New("price must be greater than zero")
	}
	if incoming.Volume < 0 {
		return domain.Tick{}, errors.New("volume must be non-negative")
	}
	if strings.TrimSpace(incoming.EventTime) == "" {
		return domain.Tick{}, errors.New("eventTime is required")
	}
	eventTime, err := time.Parse(time.RFC3339, incoming.EventTime)
	if err != nil {
		return domain.Tick{}, err
	}
	if incoming.Source == "" {
		incoming.Source = "unknown"
	}
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	return domain.Tick{
		Symbol:        incoming.Symbol,
		Price:         incoming.Price,
		Volume:        incoming.Volume,
		Source:        incoming.Source,
		EventTime:     eventTime.UTC(),
		ReceivedAt:    time.Now().UTC(),
		CorrelationID: correlationID,
	}, nil
}
