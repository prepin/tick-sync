package googletasksync

import "time"

type GoogleTask struct {
	ID      string
	Title   string
	Notes   string
	Due     string
	Updated string
}

type TickTickTaskInput struct {
	Title              string
	Details            string
	Due                string
	SourceGoogleTaskID string
}

type TickTickTask struct {
	ID string
}

type SyncedTaskRecord struct {
	GoogleTaskID   string
	GoogleUpdated  string
	GoogleTitle    string
	TickTickTaskID string
	PostSyncAction PostSyncAction
	SyncedAt       time.Time
}

type SyncSummary struct {
	Seen      int
	Created   int
	Skipped   int
	Failed    int
	Completed int
	Deleted   int
	Errors    []error
}
