package service

import "testing"

func TestParseAndValidateAcceptsValidTick(t *testing.T) {
	tick, err := parseAndValidate([]byte(`{"symbol":"aapl","price":184.52,"volume":1200,"source":"test","eventTime":"2026-05-30T12:00:00Z"}`), "corr-1")
	if err != nil {
		t.Fatalf("expected valid tick: %v", err)
	}
	if tick.Symbol != "AAPL" || tick.Price != 184.52 || tick.Volume != 1200 || tick.CorrelationID != "corr-1" {
		t.Fatalf("unexpected tick: %#v", tick)
	}
}

func TestParseAndValidateRejectsInvalidTick(t *testing.T) {
	cases := []string{
		`{"symbol":"","price":184.52,"volume":1200,"source":"test","eventTime":"2026-05-30T12:00:00Z"}`,
		`{"symbol":"AAPL","price":0,"volume":1200,"source":"test","eventTime":"2026-05-30T12:00:00Z"}`,
		`{"symbol":"AAPL","price":184.52,"volume":-1,"source":"test","eventTime":"2026-05-30T12:00:00Z"}`,
		`{"symbol":"AAPL","price":184.52,"volume":1200,"source":"test","eventTime":""}`,
		`{"symbol":"AAPL","price":184.52,"volume":1200,"source":"test","eventTime":"not-a-time"}`,
	}
	for _, payload := range cases {
		if _, err := parseAndValidate([]byte(payload), "corr-1"); err == nil {
			t.Fatalf("expected invalid tick for payload %s", payload)
		}
	}
}
