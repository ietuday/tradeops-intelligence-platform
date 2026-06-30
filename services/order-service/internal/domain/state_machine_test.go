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

func testTime() time.Time {
	return time.Unix(1700000000, 0).UTC()
}
