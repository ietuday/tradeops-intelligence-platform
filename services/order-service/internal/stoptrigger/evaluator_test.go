package stoptrigger

import (
	"testing"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
)

func TestShouldTrigger(t *testing.T) {
	tests := []struct {
		name           string
		side           string
		orderType      string
		stopPrice      float64
		referencePrice float64
		want           bool
	}{
		{"buy stop crosses above", domain.SideBuy, domain.OrderTypeStop, 100, 100, true},
		{"buy stop below stop", domain.SideBuy, domain.OrderTypeStop, 100, 99.99, false},
		{"sell stop crosses below", domain.SideSell, domain.OrderTypeStop, 100, 100, true},
		{"sell stop above stop", domain.SideSell, domain.OrderTypeStop, 100, 100.01, false},
		{"buy stop limit crosses above", domain.SideBuy, domain.OrderTypeStopLimit, 100, 101, true},
		{"sell stop limit crosses below", domain.SideSell, domain.OrderTypeStopLimit, 100, 99, true},
		{"invalid side", "HOLD", domain.OrderTypeStop, 100, 101, false},
		{"invalid order type", domain.SideBuy, domain.OrderTypeLimit, 100, 101, false},
		{"zero stop price", domain.SideBuy, domain.OrderTypeStop, 0, 101, false},
		{"negative reference price", domain.SideBuy, domain.OrderTypeStop, 100, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldTrigger(tt.side, tt.orderType, tt.stopPrice, tt.referencePrice); got != tt.want {
				t.Fatalf("ShouldTrigger() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestActivatedOrderType(t *testing.T) {
	if got := ActivatedOrderType(domain.OrderTypeStop); got != domain.OrderTypeMarket {
		t.Fatalf("STOP activates as %s", got)
	}
	if got := ActivatedOrderType(domain.OrderTypeStopLimit); got != domain.OrderTypeLimit {
		t.Fatalf("STOP_LIMIT activates as %s", got)
	}
	if got := ActivatedOrderType(domain.OrderTypeLimit); got != "" {
		t.Fatalf("LIMIT should not activate, got %s", got)
	}
}
