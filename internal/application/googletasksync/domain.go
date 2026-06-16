package googletasksync

// PostSyncAction defines what the sync use case does with a Google task after
// it has been successfully copied to TickTick and recorded locally.
type PostSyncAction string

const (
	PostSyncActionComplete PostSyncAction = "complete"
	PostSyncActionDelete   PostSyncAction = "delete"
)
