package expiry

import (
	"context"
	"errors"
	"time"
)

type TradingCalendar interface {
	ExpiryForDayOrder(ctx context.Context, submittedAt time.Time) (time.Time, error)
	IsTradingDay(ctx context.Context, date time.Time) (bool, error)
}

// WeekdayTradingCalendar models regular Monday-Friday sessions only. It relies
// on Go timezone data for DST and intentionally does not model holidays,
// half-days, or exchange-specific special sessions yet.
type WeekdayTradingCalendar struct {
	location  *time.Location
	closeHour int
	closeMin  int
}

func NewWeekdayTradingCalendar(timezone, closeTime string) (*WeekdayTradingCalendar, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	parsed, err := time.Parse("15:04", closeTime)
	if err != nil {
		return nil, err
	}
	return &WeekdayTradingCalendar{
		location:  loc,
		closeHour: parsed.Hour(),
		closeMin:  parsed.Minute(),
	}, nil
}

func (c *WeekdayTradingCalendar) ExpiryForDayOrder(ctx context.Context, submittedAt time.Time) (time.Time, error) {
	if c == nil || c.location == nil {
		return time.Time{}, errors.New("trading calendar is not configured")
	}
	local := submittedAt.In(c.location)
	expiry := c.closeFor(local)
	if !expiry.After(local) {
		expiry = c.closeFor(local.AddDate(0, 0, 1))
	}
	for {
		if err := ctx.Err(); err != nil {
			return time.Time{}, err
		}
		trading, err := c.IsTradingDay(ctx, expiry)
		if err != nil {
			return time.Time{}, err
		}
		if trading {
			return expiry.UTC(), nil
		}
		expiry = c.closeFor(expiry.AddDate(0, 0, 1))
	}
}

func (c *WeekdayTradingCalendar) IsTradingDay(_ context.Context, date time.Time) (bool, error) {
	if c == nil || c.location == nil {
		return false, errors.New("trading calendar is not configured")
	}
	weekday := date.In(c.location).Weekday()
	return weekday != time.Saturday && weekday != time.Sunday, nil
}

func (c *WeekdayTradingCalendar) closeFor(local time.Time) time.Time {
	return time.Date(local.Year(), local.Month(), local.Day(), c.closeHour, c.closeMin, 0, 0, c.location)
}
