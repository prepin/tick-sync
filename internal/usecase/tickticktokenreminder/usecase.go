// Package tickticktokenreminder creates TickTick reminders for token refresh.
package tickticktokenreminder

import "time"

const (
	mediumPriority        = 3
	refreshReminderWindow = 14 * 24 * time.Hour
)

// UseCase creates one TickTick reminder task for a stored token that expires soon.
type UseCase struct {
	tokens   TokenRepository
	ticktick TickTickGateway
	now      func() time.Time
}

// Option configures the use case.
type Option func(*UseCase)

// WithNow overrides the clock used by the use case.
func WithNow(now func() time.Time) Option {
	return func(u *UseCase) {
		u.now = now
	}
}

// New creates a TickTick token reminder use case.
func New(tokens TokenRepository, ticktick TickTickGateway, opts ...Option) *UseCase {
	u := &UseCase{
		tokens:   tokens,
		ticktick: ticktick,
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}
