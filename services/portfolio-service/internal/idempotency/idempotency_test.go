package idempotency

import (
	"testing"

	"github.com/ietuday/tradeops-intelligence-platform/services/portfolio-service/internal/domain"
)

func TestBuildIdempotencyKey(t *testing.T) {
	tests := []struct {
		name  string
		event domain.TradeExecutedEvent
		want  string
	}{
		{name: "execution first", event: domain.TradeExecutedEvent{ExecutionID: "exec-1", TradeID: "trade-1", EventID: "evt-1"}, want: "execution:exec-1"},
		{name: "trade fallback", event: domain.TradeExecutedEvent{TradeID: "trade-1", EventID: "evt-1"}, want: "trade:trade-1"},
		{name: "event fallback", event: domain.TradeExecutedEvent{EventID: "evt-1"}, want: "event:evt-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildIdempotencyKey(tt.event); got != tt.want {
				t.Fatalf("BuildIdempotencyKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
