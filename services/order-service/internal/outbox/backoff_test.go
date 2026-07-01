package outbox

import (
	"math/rand"
	"testing"
	"time"
)

func TestBackoffDelayProgressionAndCap(t *testing.T) {
	b, err := NewBackoff(time.Second, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	b.JitterPct = 0

	if got := b.Delay(0); got != time.Second {
		t.Fatalf("first retry = %s, want 1s", got)
	}
	if got := b.Delay(1); got != 2*time.Second {
		t.Fatalf("second retry = %s, want 2s", got)
	}
	if got := b.Delay(10); got != 5*time.Second {
		t.Fatalf("capped retry = %s, want 5s", got)
	}
}

func TestBackoffJitterBounds(t *testing.T) {
	b, err := NewBackoff(time.Second, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	b.JitterPct = 0.2
	b.rand = rand.New(rand.NewSource(1))

	for i := 0; i < 100; i++ {
		delay := b.Delay(2)
		if delay < 3200*time.Millisecond || delay > 4800*time.Millisecond {
			t.Fatalf("jittered delay %s outside 20%% bounds around 4s", delay)
		}
	}
}

func TestBackoffHighAttemptsDoNotOverflow(t *testing.T) {
	b, err := NewBackoff(time.Second, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	b.JitterPct = 0
	if got := b.Delay(10_000); got != time.Minute {
		t.Fatalf("high attempt delay = %s, want cap", got)
	}
}

func TestBackoffRejectsInvalidConfig(t *testing.T) {
	if _, err := NewBackoff(0, time.Second); err == nil {
		t.Fatal("expected zero base backoff to fail")
	}
	if _, err := NewBackoff(time.Second, time.Millisecond); err == nil {
		t.Fatal("expected max below base to fail")
	}
}
