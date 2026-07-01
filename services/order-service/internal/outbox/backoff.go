package outbox

import (
	"errors"
	"math/rand"
	"time"
)

type Backoff struct {
	Base      time.Duration
	Max       time.Duration
	JitterPct float64
	rand      *rand.Rand
}

func NewBackoff(base, max time.Duration) (Backoff, error) {
	if base <= 0 {
		return Backoff{}, errors.New("base backoff must be positive")
	}
	if max < base {
		return Backoff{}, errors.New("max backoff must be greater than or equal to base backoff")
	}
	return Backoff{
		Base:      base,
		Max:       max,
		JitterPct: 0.20,
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func (b Backoff) Delay(attemptCount int) time.Duration {
	if attemptCount < 0 {
		attemptCount = 0
	}
	delay := b.Base
	for i := 0; i < attemptCount; i++ {
		if delay >= b.Max/2 {
			delay = b.Max
			break
		}
		delay *= 2
	}
	if delay > b.Max {
		delay = b.Max
	}
	if b.JitterPct <= 0 {
		return delay
	}
	jitterRange := int64(float64(delay) * b.JitterPct)
	if jitterRange <= 0 {
		return delay
	}
	source := b.rand
	if source == nil {
		source = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	jitter := time.Duration(source.Int63n(jitterRange*2+1) - jitterRange)
	withJitter := delay + jitter
	if withJitter < 0 {
		return 0
	}
	if withJitter > b.Max {
		return b.Max
	}
	return withJitter
}
