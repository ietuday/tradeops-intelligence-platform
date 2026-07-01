package risk

import "errors"

var (
	ErrRejected        = errors.New("pre-trade risk rejected")
	ErrUnavailable     = errors.New("pre-trade risk unavailable")
	ErrTimeout         = errors.New("pre-trade risk timeout")
	ErrInvalidResponse = errors.New("pre-trade risk invalid response")
)

type DecisionError struct {
	Kind     error
	Decision PreTradeRiskDecision
}

func (e DecisionError) Error() string {
	if e.Decision.ReasonMessage != "" {
		return e.Decision.ReasonMessage
	}
	return e.Kind.Error()
}

func (e DecisionError) Unwrap() error {
	return e.Kind
}
