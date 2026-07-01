package service

import (
	"errors"
	"testing"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/repository"
)

func TestCanView(t *testing.T) {
	if !canView([]string{"viewer"}) {
		t.Fatal("viewer should be allowed")
	}
	if canView([]string{"unknown"}) {
		t.Fatal("unknown role should not be allowed")
	}
}

func TestNormalizeTradeExecutedValidation(t *testing.T) {
	event := domain.TradeExecutedEvent{
		EventID:           "evt-1",
		EventType:         "trade.executed",
		EventVersion:      "1.0",
		TenantID:          "tenant-1",
		ExecutionID:       "exec-1",
		BuyOrderID:        "buy-1",
		SellOrderID:       "sell-1",
		BuyerUserID:       "buyer-1",
		SellerUserID:      "seller-1",
		Symbol:            " aapl ",
		ExecutionQuantity: 10,
		ExecutionPrice:    100,
		OccurredAt:        time.Now().UTC(),
	}

	if err := normalizeTradeExecuted(&event); err != nil {
		t.Fatalf("expected valid event: %v", err)
	}
	if event.Symbol != "AAPL" || event.Currency != "USD" {
		t.Fatalf("unexpected normalization: %+v", event)
	}
}

func TestNormalizeTradeExecutedRejectsSelfTrade(t *testing.T) {
	event := domain.TradeExecutedEvent{
		EventID:           "evt-1",
		EventType:         "trade.executed",
		EventVersion:      "1.0",
		TenantID:          "tenant-1",
		ExecutionID:       "exec-1",
		BuyOrderID:        "buy-1",
		SellOrderID:       "sell-1",
		BuyerUserID:       "user-1",
		SellerUserID:      "user-1",
		Symbol:            "AAPL",
		ExecutionQuantity: 10,
		ExecutionPrice:    100,
		OccurredAt:        time.Now().UTC(),
	}

	if err := normalizeTradeExecuted(&event); !errors.Is(err, repository.ErrSelfTrade) {
		t.Fatalf("expected self-trade error, got %v", err)
	}
}
