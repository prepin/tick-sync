// Package googletasksync implements the Google Tasks to TickTick sync use case.
package googletasksync

// PostSyncAction defines what the sync use case does with a Google task after
// it has been successfully copied to TickTick and recorded locally.
type PostSyncAction string

const (
	// PostSyncActionComplete marks the Google task as completed after sync.
	PostSyncActionComplete PostSyncAction = "complete"
	// PostSyncActionDelete deletes the Google task after sync.
	PostSyncActionDelete PostSyncAction = "delete"
)
