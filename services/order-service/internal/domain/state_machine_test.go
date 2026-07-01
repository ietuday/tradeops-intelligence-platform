package domain

import (
	"testing"
	"time"
)

func TestStateTransitions(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		next    func(*Order) error
		wantErr bool
	}{
		{name: "created to validated", status: StatusCreated, next: (*Order).Validate},
		{name: "validated to accepted", status: StatusValidated, next: (*Order).Accept},
		{name: "accepted to cancelled", status: StatusAccepted, next: func(o *Order) error { return o.Cancel(testTime()) }},
		{name: "partial to cancelled", status: StatusPartiallyFilled, next: func(o *Order) error { return o.Cancel(testTime()) }},
		{name: "accepted to expired", status: StatusAccepted, next: func(o *Order) error { return o.Expire(testTime()) }},
		{name: "partial to expired", status: StatusPartiallyFilled, next: func(o *Order) error { return o.Expire(testTime()) }},
		{name: "filled to expired rejected", status: StatusFilled, next: func(o *Order) error { return o.Expire(testTime()) }, wantErr: true},
		{name: "cancelled to expired rejected", status: StatusCancelled, next: func(o *Order) error { return o.Expire(testTime()) }, wantErr: true},
		{name: "rejected to expired rejected", status: StatusRejected, next: func(o *Order) error { return o.Expire(testTime()) }, wantErr: true},
		{name: "expired to expired rejected", status: StatusExpired, next: func(o *Order) error { return o.Expire(testTime()) }, wantErr: true},
		{name: "filled to cancelled rejected", status: StatusFilled, next: func(o *Order) error { return o.Cancel(testTime()) }, wantErr: true},
		{name: "cancelled to accepted rejected", status: StatusCancelled, next: (*Order).Accept, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			order := &Order{Status: tc.status, Quantity: 10, RemainingQuantity: 10}
			err := tc.next(order)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExpirePreservesRemainingQuantity(t *testing.T) {
	order := &Order{Status: StatusPartiallyFilled, Quantity: 100, FilledQuantity: 40, RemainingQuantity: 60}
	at := testTime()
	if err := order.Expire(at); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != StatusExpired {
		t.Fatalf("status = %s, want %s", order.Status, StatusExpired)
	}
	if order.RemainingQuantity != 60 {
		t.Fatalf("remaining quantity = %v, want 60", order.RemainingQuantity)
	}
	if order.ExpiredAt == nil || !order.ExpiredAt.Equal(at) {
		t.Fatalf("expiredAt = %v, want %v", order.ExpiredAt, at)
	}
}

func testTime() time.Time {
	return time.Unix(1700000000, 0).UTC()
}
