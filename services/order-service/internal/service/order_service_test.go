package service

import (
	"testing"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
)

func TestValidateOrder(t *testing.T) {
	limit := 10.0
	stop := 9.0
	cases := []struct {
		name  string
		order domain.Order
		valid bool
	}{
		{name: "market", order: domain.Order{Symbol: "AAPL", Side: "BUY", OrderType: "MARKET", Quantity: 1}, valid: true},
		{name: "limit", order: domain.Order{Symbol: "AAPL", Side: "SELL", OrderType: "LIMIT", Quantity: 1, LimitPrice: &limit}, valid: true},
		{name: "stop", order: domain.Order{Symbol: "AAPL", Side: "SELL", OrderType: "STOP_LOSS", Quantity: 1, StopPrice: &stop}, valid: true},
		{name: "missing symbol", order: domain.Order{Side: "BUY", OrderType: "MARKET", Quantity: 1}, valid: false},
		{name: "bad side", order: domain.Order{Symbol: "AAPL", Side: "HOLD", OrderType: "MARKET", Quantity: 1}, valid: false},
		{name: "bad type", order: domain.Order{Symbol: "AAPL", Side: "BUY", OrderType: "ICEBERG", Quantity: 1}, valid: false},
		{name: "bad quantity", order: domain.Order{Symbol: "AAPL", Side: "BUY", OrderType: "MARKET", Quantity: 0}, valid: false},
		{name: "missing limit", order: domain.Order{Symbol: "AAPL", Side: "BUY", OrderType: "LIMIT", Quantity: 1}, valid: false},
		{name: "missing stop", order: domain.Order{Symbol: "AAPL", Side: "BUY", OrderType: "STOP_LOSS", Quantity: 1}, valid: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errText := validateOrder(tc.order)
			if tc.valid && errText != "" {
				t.Fatalf("expected valid order, got %s", errText)
			}
			if !tc.valid && errText == "" {
				t.Fatal("expected invalid order")
			}
		})
	}
}

func TestHashRequestStable(t *testing.T) {
	body := []byte(`{"symbol":"AAPL","side":"BUY"}`)
	if hashRequest(body) != hashRequest(body) {
		t.Fatal("expected stable request hash")
	}
	if hashRequest(body) == hashRequest([]byte(`{"symbol":"MSFT","side":"BUY"}`)) {
		t.Fatal("expected different payload hash")
	}
}
