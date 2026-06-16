package googletasksync

// SyncGoogleTasksToTickTickResult is the output envelope of the sync use case.
type SyncGoogleTasksToTickTickResult struct {
	Seen      int
	Created   int
	Skipped   int
	Failed    int
	Completed int
	Deleted   int
	Errors    []error
}
