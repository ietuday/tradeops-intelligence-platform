package expiry

import (
	"context"
	"testing"
	"time"
)

func TestWeekdayTradingCalendarExpiryForDayOrder(t *testing.T) {
	calendar, err := NewWeekdayTradingCalendar("America/New_York", "16:00")
	if err != nil {
		t.Fatalf("calendar: %v", err)
	}
	cases := []struct {
		name      string
		submitted string
		wantLocal string
	}{
		{name: "before close same day", submitted: "2026-07-01T15:30:00-04:00", wantLocal: "2026-07-01T16:00:00-04:00"},
		{name: "exactly at close next weekday", submitted: "2026-07-01T16:00:00-04:00", wantLocal: "2026-07-02T16:00:00-04:00"},
		{name: "after close next weekday", submitted: "2026-07-01T17:00:00-04:00", wantLocal: "2026-07-02T16:00:00-04:00"},
		{name: "friday after close rolls to monday", submitted: "2026-07-03T17:00:00-04:00", wantLocal: "2026-07-06T16:00:00-04:00"},
		{name: "saturday rolls to monday", submitted: "2026-07-04T12:00:00-04:00", wantLocal: "2026-07-06T16:00:00-04:00"},
		{name: "sunday rolls to monday", submitted: "2026-07-05T12:00:00-04:00", wantLocal: "2026-07-06T16:00:00-04:00"},
		{name: "dst uses local close", submitted: "2026-11-02T12:00:00-05:00", wantLocal: "2026-11-02T16:00:00-05:00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			submitted, err := time.Parse(time.RFC3339, tc.submitted)
			if err != nil {
				t.Fatal(err)
			}
			want, err := time.Parse(time.RFC3339, tc.wantLocal)
			if err != nil {
				t.Fatal(err)
			}
			got, err := calendar.ExpiryForDayOrder(context.Background(), submitted)
			if err != nil {
				t.Fatalf("expiry: %v", err)
			}
			if !got.Equal(want.UTC()) {
				t.Fatalf("expiry = %s, want %s", got, want.UTC())
			}
			if got.Location() != time.UTC {
				t.Fatalf("expiry location = %s, want UTC", got.Location())
			}
		})
	}
}

func TestWeekdayTradingCalendarValidation(t *testing.T) {
	if _, err := NewWeekdayTradingCalendar("No/SuchZone", "16:00"); err == nil {
		t.Fatal("expected invalid timezone error")
	}
	if _, err := NewWeekdayTradingCalendar("America/New_York", "4pm"); err == nil {
		t.Fatal("expected invalid close time error")
	}
}
