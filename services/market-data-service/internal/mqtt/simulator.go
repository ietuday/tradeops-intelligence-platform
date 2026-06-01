package mqtt

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

var symbols = []string{"AAPL", "TSLA", "MSFT", "BTC-USD", "ETH-USD", "NIFTY50", "BANKNIFTY"}

type Simulator struct {
	client   paho.Client
	interval time.Duration
	logger   *slog.Logger
	rand     *rand.Rand
}

func NewSimulator(client paho.Client, interval time.Duration, logger *slog.Logger) *Simulator {
	return &Simulator{
		client:   client,
		interval: interval,
		logger:   logger,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Simulator) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.publish()
			}
		}
	}()
}

func (s *Simulator) publish() {
	symbol := symbols[s.rand.Intn(len(symbols))]
	payload := map[string]any{
		"symbol":    symbol,
		"price":     s.price(symbol),
		"volume":    s.rand.Intn(5000),
		"source":    "local-simulator",
		"eventTime": time.Now().UTC().Format(time.RFC3339),
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.Warn("failed to marshal simulated tick", "error", err)
		return
	}
	topic := "market/" + symbol + "/tick"
	token := s.client.Publish(topic, 1, false, bytes)
	if token.WaitTimeout(2*time.Second) && token.Error() != nil {
		s.logger.Warn("failed to publish simulated tick", "topic", topic, "error", token.Error())
	}
}

func (s *Simulator) price(symbol string) float64 {
	switch symbol {
	case "BTC-USD":
		return 60000 + s.rand.Float64()*5000
	case "ETH-USD":
		return 2500 + s.rand.Float64()*500
	case "NIFTY50":
		return 22000 + s.rand.Float64()*500
	case "BANKNIFTY":
		return 47000 + s.rand.Float64()*1000
	default:
		return 100 + s.rand.Float64()*300
	}
}
