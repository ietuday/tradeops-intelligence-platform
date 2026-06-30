package matching

import (
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
)

func TestLimitOrderPriceTimePriorityAndPartialFill(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	price10 := 10.0
	price9 := 9.0
	incoming := FromDomain(domain.Order{ID: "buy-1", UserID: "buyer", Symbol: "AAPL", Side: domain.SideBuy, OrderType: domain.OrderTypeLimit, TimeInForce: domain.TimeInForceGTC, Quantity: 12, RemainingQuantity: 12, LimitPrice: &price10, CreatedAt: now})
	resting := []Order{
		FromDomain(domain.Order{ID: "sell-late", UserID: "seller-2", Symbol: "AAPL", Side: domain.SideSell, OrderType: domain.OrderTypeLimit, Quantity: 10, RemainingQuantity: 10, LimitPrice: &price10, CreatedAt: now.Add(2 * time.Second)}),
		FromDomain(domain.Order{ID: "sell-better", UserID: "seller-3", Symbol: "AAPL", Side: domain.SideSell, OrderType: domain.OrderTypeLimit, Quantity: 5, RemainingQuantity: 5, LimitPrice: &price9, CreatedAt: now.Add(3 * time.Second)}),
		FromDomain(domain.Order{ID: "sell-early", UserID: "seller-1", Symbol: "AAPL", Side: domain.SideSell, OrderType: domain.OrderTypeLimit, Quantity: 10, RemainingQuantity: 10, LimitPrice: &price10, CreatedAt: now.Add(time.Second)}),
	}
	result := NewEngine().Match(incoming, resting, now)
	if len(result.Executions) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(result.Executions))
	}
	if result.Executions[0].SellOrderID != "sell-better" {
		t.Fatalf("expected best price first, got %s", result.Executions[0].SellOrderID)
	}
	if result.Executions[1].SellOrderID != "sell-early" {
		t.Fatalf("expected FIFO at equal price, got %s", result.Executions[1].SellOrderID)
	}
	if got := RatFloat(result.Incoming.RemainingQuantity); got != 0 {
		t.Fatalf("expected incoming filled, remaining %v", got)
	}
}

func TestFOKNoFillDoesNotMutateResting(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	price := 10.0
	incoming := FromDomain(domain.Order{ID: "buy-1", UserID: "buyer", Symbol: "AAPL", Side: domain.SideBuy, OrderType: domain.OrderTypeLimit, TimeInForce: domain.TimeInForceFOK, Quantity: 20, RemainingQuantity: 20, LimitPrice: &price, CreatedAt: now})
	resting := []Order{FromDomain(domain.Order{ID: "sell-1", UserID: "seller", Symbol: "AAPL", Side: domain.SideSell, OrderType: domain.OrderTypeLimit, Quantity: 5, RemainingQuantity: 5, LimitPrice: &price, CreatedAt: now})}
	result := NewEngine().Match(incoming, resting, now)
	if len(result.Executions) != 0 {
		t.Fatalf("expected no executions, got %d", len(result.Executions))
	}
	if !result.Cancelled {
		t.Fatal("expected FOK to cancel when full fill is unavailable")
	}
	if got := RatFloat(resting[0].RemainingQuantity); got != 5 {
		t.Fatalf("resting order mutated, remaining %v", got)
	}
}

func TestIOCRemainderCancelled(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	price := 10.0
	incoming := FromDomain(domain.Order{ID: "buy-1", UserID: "buyer", Symbol: "AAPL", Side: domain.SideBuy, OrderType: domain.OrderTypeLimit, TimeInForce: domain.TimeInForceIOC, Quantity: 20, RemainingQuantity: 20, LimitPrice: &price, CreatedAt: now})
	resting := []Order{FromDomain(domain.Order{ID: "sell-1", UserID: "seller", Symbol: "AAPL", Side: domain.SideSell, OrderType: domain.OrderTypeLimit, Quantity: 5, RemainingQuantity: 5, LimitPrice: &price, CreatedAt: now})}
	result := NewEngine().Match(incoming, resting, now)
	if len(result.Executions) != 1 {
		t.Fatalf("expected one execution, got %d", len(result.Executions))
	}
	if !result.Cancelled || result.Rested {
		t.Fatal("expected IOC remainder cancellation without resting")
	}
	if got := RatFloat(result.Incoming.RemainingQuantity); got != 15 {
		t.Fatalf("expected 15 remaining before cancellation persistence, got %v", got)
	}
}
