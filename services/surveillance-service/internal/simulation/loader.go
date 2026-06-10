package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
)

type HistoricalEventLoader interface {
	Load(ctx context.Context, tenantID string, lookback time.Duration) ([]domain.SourceEvent, error)
}

type DemoEventLoader struct{}

func NewDemoEventLoader() DemoEventLoader {
	return DemoEventLoader{}
}

func (DemoEventLoader) Load(ctx context.Context, tenantID string, lookback time.Duration) ([]domain.SourceEvent, error) {
	_ = ctx
	start := time.Now().UTC().Add(-lookback)
	events := []domain.SourceEvent{
		sourceEvent("order.filled", "sim-large-1", start.Add(5*time.Minute), map[string]any{"tenantId": tenantID, "orderId": "sim-large-1", "userId": "demo-risk-user", "symbol": "AAPL", "price": 210.5, "quantity": 200, "notional": 42100.0}),
		sourceEvent("order.filled", "sim-large-2", start.Add(10*time.Minute), map[string]any{"tenantId": tenantID, "orderId": "sim-large-2", "userId": "demo-risk-user", "symbol": "MSFT", "price": 420.0, "quantity": 400, "notional": 168000.0}),
		sourceEvent("order.filled", "sim-large-3", start.Add(15*time.Minute), map[string]any{"tenantId": tenantID, "orderId": "sim-large-3", "userId": "demo-risk-user", "symbol": "NVDA", "price": 950.0, "quantity": 300, "notional": 285000.0}),
		sourceEvent("risk.score.updated", "sim-risk-1", start.Add(20*time.Minute), map[string]any{"tenantId": tenantID, "portfolioId": "portfolio-demo", "userId": "demo-risk-user", "score": 72.0}),
		sourceEvent("risk.score.updated", "sim-risk-2", start.Add(25*time.Minute), map[string]any{"tenantId": tenantID, "portfolioId": "portfolio-demo", "userId": "demo-risk-user", "score": 86.0}),
		sourceEvent("market.ticks", "sim-price-1", start.Add(30*time.Minute), map[string]any{"tenantId": tenantID, "symbol": "AAPL", "price": 100.0, "volume": 1000}),
		sourceEvent("market.ticks", "sim-price-2", start.Add(31*time.Minute), map[string]any{"tenantId": tenantID, "symbol": "AAPL", "price": 112.5, "volume": 2300}),
	}
	for i := 0; i < 7; i++ {
		events = append(events, sourceEvent("order.created", fmt.Sprintf("sim-rapid-%d", i+1), start.Add(time.Duration(35+i)*time.Minute), map[string]any{"tenantId": tenantID, "orderId": fmt.Sprintf("sim-rapid-%d", i+1), "userId": "rapid-demo-user", "symbol": "TSLA", "price": 250.0, "quantity": 10}))
	}
	for i := 0; i < 5; i++ {
		events = append(events, sourceEvent("order.cancelled", fmt.Sprintf("sim-cancel-%d", i+1), start.Add(time.Duration(45+i)*time.Minute), map[string]any{"tenantId": tenantID, "orderId": fmt.Sprintf("sim-cancel-%d", i+1), "userId": "cancel-demo-user", "symbol": "AMD"}))
	}
	return events, nil
}

func sourceEvent(topic, key string, ts time.Time, payload map[string]any) domain.SourceEvent {
	payload["occurredAt"] = ts.Format(time.RFC3339)
	data, _ := json.Marshal(payload)
	return domain.SourceEvent{Topic: topic, Key: key, Value: data, CorrelationID: "rule-simulation-demo"}
}
