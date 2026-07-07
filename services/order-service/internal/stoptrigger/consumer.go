package stoptrigger

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/observability"
	"github.com/segmentio/kafka-go"
)

type PriceConsumer struct {
	reader  *kafka.Reader
	store   ReferencePriceStore
	metrics *observability.Metrics
	logger  *slog.Logger
	mu      sync.RWMutex
	running bool
}

func NewPriceConsumer(brokers []string, topic, groupID string, store ReferencePriceStore, metrics *observability.Metrics, logger *slog.Logger) *PriceConsumer {
	return &PriceConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
		store:   store,
		metrics: metrics,
		logger:  logger,
	}
}

func (c *PriceConsumer) Run(ctx context.Context) error {
	c.setRunning(true)
	defer c.setRunning(false)
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return nil
			}
			c.metrics.StopTriggerErrors.WithLabelValues("market_consumer_error").Inc()
			c.logger.Warn("market price consumer fetch failed", "error", err)
			continue
		}
		if err := c.handle(ctx, msg); err != nil {
			c.metrics.StopTriggerErrors.WithLabelValues("market_event_error").Inc()
			c.logger.Warn("market price event rejected", "error", err)
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.metrics.StopTriggerErrors.WithLabelValues("market_commit_error").Inc()
			c.logger.Warn("market price consumer commit failed", "error", err)
		}
	}
}

func (c *PriceConsumer) Close() error {
	return c.reader.Close()
}

func (c *PriceConsumer) handle(ctx context.Context, msg kafka.Message) error {
	event, err := parseMarketPriceEvent(msg.Value)
	if err != nil {
		return err
	}
	if err := c.store.UpsertMarketReferencePrice(ctx, event.TenantID, event.Symbol, event.Price, event.Source, event.EventTime); err != nil {
		return err
	}
	c.metrics.ReferencePriceAge.WithLabelValues(event.Symbol).Set(time.Since(event.EventTime.UTC()).Seconds())
	return nil
}

func (c *PriceConsumer) Running() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *PriceConsumer) setRunning(running bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = running
}

func parseMarketPriceEvent(payload []byte) (MarketPriceUpdated, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return MarketPriceUpdated{}, err
	}
	eventType, _ := raw["eventType"].(string)
	if eventType != "" && eventType != "market.price.updated" && eventType != "market.tick.received" && eventType != "market.ticks" && eventType != "market.tick.normalized" {
		return MarketPriceUpdated{}, errors.New("unsupported market event type")
	}
	tenantID, _ := raw["tenantId"].(string)
	if strings.TrimSpace(tenantID) == "" {
		tenantID = "default-tenant"
	}
	symbol, _ := raw["symbol"].(string)
	source, _ := raw["source"].(string)
	if strings.TrimSpace(source) == "" {
		source = "market-data-service"
	}
	correlationID, _ := raw["correlationId"].(string)
	price, err := numeric(raw["price"])
	if err != nil {
		return MarketPriceUpdated{}, err
	}
	eventTime := time.Now().UTC()
	if value, _ := raw["eventTime"].(string); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return MarketPriceUpdated{}, err
		}
		eventTime = parsed.UTC()
	}
	event := MarketPriceUpdated{
		EventType:     eventType,
		EventVersion:  stringValue(raw["eventVersion"]),
		TenantID:      strings.TrimSpace(tenantID),
		Symbol:        strings.ToUpper(strings.TrimSpace(symbol)),
		Price:         price,
		Source:        strings.TrimSpace(source),
		EventTime:     eventTime,
		CorrelationID: strings.TrimSpace(correlationID),
	}
	if event.Symbol == "" {
		return MarketPriceUpdated{}, errors.New("symbol is required")
	}
	if event.Price <= 0 {
		return MarketPriceUpdated{}, errors.New("price must be positive")
	}
	return event, nil
}

func numeric(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	default:
		return 0, errors.New("price is required")
	}
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
