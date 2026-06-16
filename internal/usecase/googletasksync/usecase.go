package googletasksync

import "time"

// SyncGoogleTasksToTickTickUseCase copies uncompleted Google tasks into TickTick
// and completes or deletes the source tasks.
type SyncGoogleTasksToTickTickUseCase struct {
	google            GoogleTasksGateway
	ticktick          TickTickGateway
	repo              SyncedTaskRepository
	postSyncAction    PostSyncAction
	delayTodayImports bool
	location          *time.Location
	now               func() time.Time
}

// Option configures the use case.
type Option func(*SyncGoogleTasksToTickTickUseCase)

// WithClock sets the clock used to record sync timestamps.
func WithClock(now func() time.Time) Option {
	return func(u *SyncGoogleTasksToTickTickUseCase) {
		if now != nil {
			u.now = now
		}
	}
}

// WithTodayImportDelay delays importing Google tasks due today until they become overdue.
func WithTodayImportDelay(enabled bool) Option {
	return func(u *SyncGoogleTasksToTickTickUseCase) {
		u.delayTodayImports = enabled
	}
}

// WithLocation sets the location used for calendar-day decisions.
func WithLocation(location *time.Location) Option {
	return func(u *SyncGoogleTasksToTickTickUseCase) {
		if location != nil {
			u.location = location
		}
	}
}

// New creates a SyncGoogleTasksToTickTickUseCase.
func New(
	google GoogleTasksGateway,
	ticktick TickTickGateway,
	repo SyncedTaskRepository,
	postSyncAction PostSyncAction,
	opts ...Option,
) *SyncGoogleTasksToTickTickUseCase {
	uc := &SyncGoogleTasksToTickTickUseCase{
		google:         google,
		ticktick:       ticktick,
		repo:           repo,
		postSyncAction: postSyncAction,
		location:       time.Local,
		now:            time.Now,
	}

	for _, opt := range opts {
		opt(uc)
	}

	return uc
}
