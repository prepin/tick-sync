package googletasksync

import "time"

type Usecase struct {
	google         GoogleTasksClient
	ticktick       TickTickClient
	store          SyncStore
	postSyncAction PostSyncAction
	now            func() time.Time
}

type Option func(*Usecase)

func WithClock(now func() time.Time) Option {
	return func(u *Usecase) {
		if now != nil {
			u.now = now
		}
	}
}

func New(google GoogleTasksClient, ticktick TickTickClient, store SyncStore, postSyncAction PostSyncAction, opts ...Option) *Usecase {
	uc := &Usecase{
		google:         google,
		ticktick:       ticktick,
		store:          store,
		postSyncAction: normalizePostSyncAction(postSyncAction),
		now:            time.Now,
	}

	for _, opt := range opts {
		opt(uc)
	}

	return uc
}

func normalizePostSyncAction(action PostSyncAction) PostSyncAction {
	if action == "" {
		return PostSyncActionComplete
	}

	return action
}
