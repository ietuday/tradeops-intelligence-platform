package matching

import (
	"sort"

	"github.com/ietuday/tradeops-intelligence-platform/services/order-service/internal/domain"
)

type Book struct {
	Bids []Order
	Asks []Order
}

func NewBook(resting []Order) *Book {
	book := &Book{}
	for _, order := range resting {
		if order.Side == domain.SideBuy {
			book.Bids = append(book.Bids, order)
		} else {
			book.Asks = append(book.Asks, order)
		}
	}
	book.Sort()
	return book
}

func (b *Book) Sort() {
	sort.SliceStable(b.Bids, func(i, j int) bool {
		cmp := b.Bids[i].LimitPrice.Cmp(b.Bids[j].LimitPrice)
		if cmp == 0 {
			return b.Bids[i].CreatedAt.Before(b.Bids[j].CreatedAt)
		}
		return cmp > 0
	})
	sort.SliceStable(b.Asks, func(i, j int) bool {
		cmp := b.Asks[i].LimitPrice.Cmp(b.Asks[j].LimitPrice)
		if cmp == 0 {
			return b.Asks[i].CreatedAt.Before(b.Asks[j].CreatedAt)
		}
		return cmp < 0
	})
}

func (b *Book) RestingFor(incoming Order) []Order {
	if incoming.Side == domain.SideBuy {
		return b.Asks
	}
	return b.Bids
}
