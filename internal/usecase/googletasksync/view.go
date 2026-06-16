package googletasksync

// GoogleTaskView is a usecase-level read projection of a Google task.
type GoogleTaskView struct {
	ID      string
	Title   string
	Notes   string
	Status  string
	Due     string
	Updated string
}

// TickTickTaskView is a usecase-level read projection of a created TickTick task.
type TickTickTaskView struct {
	ID string
}
