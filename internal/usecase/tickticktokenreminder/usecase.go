// Package tickticktokenreminder creates reminders for TickTick token renewal.
package tickticktokenreminder

import "time"

const (
	mediumPriority        = 3
	refreshReminderWindow = 14 * 24 * time.Hour
)

// UseCase reminds in TickTick before token expiry and in Google Tasks after expiry.
type UseCase struct {
	tokens   TokenRepository
	ticktick TickTickGateway
	google   GoogleTasksGateway
	now      func() time.Time
	location *time.Location
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
func New(tokens TokenRepository, ticktick TickTickGateway, google GoogleTasksGateway, opts ...Option) *UseCase {
	u := &UseCase{
		tokens:   tokens,
		ticktick: ticktick,
		google:   google,
		now:      time.Now,
		location: time.Local,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// WithLocation sets the timezone used to select the reminder's next noon.
func WithLocation(location *time.Location) Option {
	return func(u *UseCase) {
		if location != nil {
			u.location = location
		}
	}
}
