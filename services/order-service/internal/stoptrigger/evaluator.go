package stoptrigger

import "github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"

func ShouldTrigger(side, orderType string, stopPrice, referencePrice float64) bool {
	if stopPrice <= 0 || referencePrice <= 0 {
		return false
	}
	if orderType != domain.OrderTypeStop && orderType != domain.OrderTypeStopLimit {
		return false
	}
	switch side {
	case domain.SideBuy:
		return referencePrice >= stopPrice
	case domain.SideSell:
		return referencePrice <= stopPrice
	default:
		return false
	}
}

func ActivatedOrderType(orderType string) string {
	switch orderType {
	case domain.OrderTypeStop:
		return domain.OrderTypeMarket
	case domain.OrderTypeStopLimit:
		return domain.OrderTypeLimit
	default:
		return ""
	}
}
