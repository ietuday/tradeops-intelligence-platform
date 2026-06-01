package service

import "testing"

func TestCanView(t *testing.T) {
	if !canView([]string{"viewer"}) {
		t.Fatal("viewer should be allowed")
	}
	if canView([]string{"unknown"}) {
		t.Fatal("unknown role should not be allowed")
	}
}
