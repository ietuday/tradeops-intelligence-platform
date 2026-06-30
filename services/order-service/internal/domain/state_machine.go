package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidTransition = errors.New("invalid order state transition")
	ErrTerminalOrder     = errors.New("terminal order cannot be changed")
)

func (o *Order) Validate() error {
	return o.transition(StatusValidated)
}

func (o *Order) SubmitForRisk() error {
	return o.transition(StatusRiskPending)
}

func (o *Order) Accept() error {
	return o.transition(StatusAccepted)
}

func (o *Order) Reject(reason string) error {
	if err := o.transition(StatusRejected); err != nil {
		return err
	}
	o.RejectReason = &reason
	return nil
}

func (o *Order) RiskReject(decisionID, reason string) error {
	if err := o.transition(StatusRiskRejected); err != nil {
		return err
	}
	o.RiskDecisionID = &decisionID
	o.RejectReason = &reason
	return nil
}

func (o *Order) ApplyFill(quantity, price float64, at time.Time) error {
	if o.IsTerminal() {
		return fmt.Errorf("%w: %s", ErrTerminalOrder, o.Status)
	}
	if o.Status != StatusAccepted && o.Status != StatusPartiallyFilled {
		return fmt.Errorf("%w: %s to fill", ErrInvalidTransition, o.Status)
	}
	previousFilled := o.FilledQuantity
	newFilled := previousFilled + quantity
	if quantity <= 0 || newFilled > o.Quantity {
		return errors.New("invalid fill quantity")
	}
	o.FilledQuantity = newFilled
	o.RemainingQuantity = o.Quantity - o.FilledQuantity
	if previousFilled == 0 || o.AverageFillPrice == nil {
		avg := price
		o.AverageFillPrice = &avg
		o.FillPrice = &avg
	} else {
		avg := ((*o.AverageFillPrice * previousFilled) + (price * quantity)) / newFilled
		o.AverageFillPrice = &avg
		o.FillPrice = &avg
	}
	o.LastExecutionAt = &at
	o.UpdatedAt = at
	if o.RemainingQuantity == 0 {
		o.Status = StatusFilled
		o.FilledAt = &at
		return nil
	}
	o.Status = StatusPartiallyFilled
	return nil
}

func (o *Order) Cancel(at time.Time) error {
	if o.Status != StatusAccepted && o.Status != StatusPartiallyFilled {
		return fmt.Errorf("%w: %s to cancelled", ErrInvalidTransition, o.Status)
	}
	o.Status = StatusCancelled
	o.RemainingQuantity = 0
	o.CancelledAt = &at
	o.UpdatedAt = at
	return nil
}

func (o *Order) Expire(at time.Time) error {
	if o.Status != StatusAccepted && o.Status != StatusPartiallyFilled {
		return fmt.Errorf("%w: %s to expired", ErrInvalidTransition, o.Status)
	}
	o.Status = StatusExpired
	o.RemainingQuantity = 0
	o.UpdatedAt = at
	return nil
}

func (o *Order) Amend(quantity *float64, limitPrice *float64, stopPrice *float64, expiresAt *time.Time, at time.Time) error {
	if o.Status != StatusAccepted && o.Status != StatusPartiallyFilled {
		return fmt.Errorf("%w: %s to amended", ErrInvalidTransition, o.Status)
	}
	if quantity != nil {
		if *quantity < o.FilledQuantity {
			return errors.New("amended quantity cannot be lower than filled quantity")
		}
		o.Quantity = *quantity
		o.RemainingQuantity = o.Quantity - o.FilledQuantity
	}
	if limitPrice != nil {
		o.LimitPrice = limitPrice
	}
	if stopPrice != nil {
		o.StopPrice = stopPrice
	}
	o.ExpiresAt = expiresAt
	o.UpdatedAt = at
	return nil
}

func (o *Order) transition(next string) error {
	if o.IsTerminal() {
		return fmt.Errorf("%w: %s", ErrTerminalOrder, o.Status)
	}
	allowed := map[string]map[string]bool{
		StatusCreated: {
			StatusValidated:    true,
			StatusRejected:     true,
			StatusRiskRejected: true,
		},
		StatusValidated: {
			StatusRiskPending:  true,
			StatusAccepted:     true,
			StatusRejected:     true,
			StatusRiskRejected: true,
		},
		StatusRiskPending: {
			StatusAccepted:     true,
			StatusRiskRejected: true,
			StatusRejected:     true,
		},
		StatusAccepted: {
			StatusPartiallyFilled: true,
			StatusFilled:          true,
			StatusCancelled:       true,
			StatusExpired:         true,
		},
		StatusPartiallyFilled: {
			StatusFilled:    true,
			StatusCancelled: true,
			StatusExpired:   true,
		},
	}
	if !allowed[o.Status][next] {
		return fmt.Errorf("%w: %s to %s", ErrInvalidTransition, o.Status, next)
	}
	o.Status = next
	return nil
}
