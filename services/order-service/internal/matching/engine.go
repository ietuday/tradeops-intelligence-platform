package matching

import (
	"math/big"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Match(incoming Order, resting []Order, now time.Time) Result {
	book := NewBook(resting)
	if incoming.TimeInForce == "" {
		incoming.TimeInForce = domain.TimeInForceDay
	}
	if incoming.TimeInForce == domain.TimeInForceFOK && !canFullyFill(incoming, book.RestingFor(incoming)) {
		return Result{Incoming: incoming, Resting: resting, Cancelled: true}
	}

	result := Result{Incoming: incoming}
	candidates := book.RestingFor(incoming)
	for i := range candidates {
		if incoming.RemainingQuantity.Sign() == 0 {
			break
		}
		maker := candidates[i]
		if !crosses(incoming, maker) {
			break
		}
		qty := minRat(incoming.RemainingQuantity, maker.RemainingQuantity)
		price := cloneRat(maker.LimitPrice)
		if qty.Sign() == 0 || price == nil {
			continue
		}
		incoming.RemainingQuantity.Sub(incoming.RemainingQuantity, qty)
		incoming.FilledQuantity.Add(incoming.FilledQuantity, qty)
		maker.RemainingQuantity.Sub(maker.RemainingQuantity, qty)
		maker.FilledQuantity.Add(maker.FilledQuantity, qty)
		result.Resting = append(result.Resting, maker)
		execution := Execution{
			Symbol:     incoming.Symbol,
			Quantity:   cloneRat(qty),
			Price:      price,
			ExecutedAt: now,
		}
		if incoming.Side == domain.SideBuy {
			execution.BuyOrderID = incoming.ID
			execution.BuyerUserID = incoming.UserID
			execution.SellOrderID = maker.ID
			execution.SellerUserID = maker.UserID
		} else {
			execution.BuyOrderID = maker.ID
			execution.BuyerUserID = maker.UserID
			execution.SellOrderID = incoming.ID
			execution.SellerUserID = incoming.UserID
		}
		result.Executions = append(result.Executions, execution)
	}
	result.Incoming = incoming
	result.Cancelled = shouldCancelRemainder(incoming)
	result.Rested = incoming.RemainingQuantity.Sign() > 0 && !result.Cancelled
	return result
}

func canFullyFill(incoming Order, resting []Order) bool {
	available := new(big.Rat)
	for _, maker := range resting {
		if !crosses(incoming, maker) {
			break
		}
		available.Add(available, maker.RemainingQuantity)
		if available.Cmp(incoming.RemainingQuantity) >= 0 {
			return true
		}
	}
	return false
}

func crosses(incoming, maker Order) bool {
	if maker.LimitPrice == nil {
		return false
	}
	if incoming.OrderType == domain.OrderTypeMarket {
		return true
	}
	if incoming.LimitPrice == nil {
		return false
	}
	if incoming.Side == domain.SideBuy {
		return maker.LimitPrice.Cmp(incoming.LimitPrice) <= 0
	}
	return maker.LimitPrice.Cmp(incoming.LimitPrice) >= 0
}

func shouldCancelRemainder(order Order) bool {
	if order.RemainingQuantity.Sign() == 0 {
		return false
	}
	return order.OrderType == domain.OrderTypeMarket ||
		order.TimeInForce == domain.TimeInForceIOC ||
		order.TimeInForce == domain.TimeInForceFOK
}

func minRat(a, b *big.Rat) *big.Rat {
	if a.Cmp(b) <= 0 {
		return cloneRat(a)
	}
	return cloneRat(b)
}
