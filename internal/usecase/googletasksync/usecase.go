package googletasksync

import "time"

// SyncGoogleTasksToTickTickUseCase copies uncompleted Google tasks into TickTick
// and completes or deletes the source tasks.
type SyncGoogleTasksToTickTickUseCase struct {
	google         GoogleTasksGateway
	ticktick       TickTickGateway
	repo           SyncedTaskRepository
	postSyncAction PostSyncAction
	now            func() time.Time
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
		now:            time.Now,
	}

	for _, opt := range opts {
		opt(uc)
	}

	return uc
}
