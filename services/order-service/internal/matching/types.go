package matching

import (
	"math/big"
	"strconv"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
)

type Order struct {
	ID                string
	TenantID          string
	UserID            string
	Symbol            string
	Side              string
	OrderType         string
	TimeInForce       string
	Quantity          *big.Rat
	FilledQuantity    *big.Rat
	RemainingQuantity *big.Rat
	LimitPrice        *big.Rat
	CreatedAt         time.Time
}

type Execution struct {
	BuyOrderID   string
	SellOrderID  string
	BuyerUserID  string
	SellerUserID string
	Symbol       string
	Quantity     *big.Rat
	Price        *big.Rat
	ExecutedAt   time.Time
}

type Result struct {
	Incoming   Order
	Resting    []Order
	Executions []Execution
	Rested     bool
	Cancelled  bool
}

func FromDomain(order domain.Order) Order {
	return Order{
		ID:                order.ID,
		TenantID:          order.TenantID,
		UserID:            order.UserID,
		Symbol:            order.Symbol,
		Side:              order.Side,
		OrderType:         order.OrderType,
		TimeInForce:       order.TimeInForce,
		Quantity:          ratFromFloat(order.Quantity),
		FilledQuantity:    ratFromFloat(order.FilledQuantity),
		RemainingQuantity: ratFromFloat(order.RemainingQuantity),
		LimitPrice:        ratFromPtr(order.LimitPrice),
		CreatedAt:         order.CreatedAt,
	}
}

func ApplyToDomain(src Order, dst *domain.Order) {
	dst.FilledQuantity = ratFloat(src.FilledQuantity)
	dst.RemainingQuantity = ratFloat(src.RemainingQuantity)
}

func RatFloat(r *big.Rat) float64 {
	return ratFloat(r)
}

func RatString(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	return r.FloatString(10)
}

func ratFromPtr(value *float64) *big.Rat {
	if value == nil {
		return nil
	}
	return ratFromFloat(*value)
}

func ratFromFloat(value float64) *big.Rat {
	rat, ok := new(big.Rat).SetString(strconv.FormatFloat(value, 'f', -1, 64))
	if !ok {
		return new(big.Rat)
	}
	return rat
}

func ratFloat(r *big.Rat) float64 {
	if r == nil {
		return 0
	}
	value, _ := r.Float64()
	return value
}

func cloneRat(r *big.Rat) *big.Rat {
	if r == nil {
		return nil
	}
	return new(big.Rat).Set(r)
}
