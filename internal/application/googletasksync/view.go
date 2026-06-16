package googletasksync

// GoogleTaskView is an application-level read projection of a Google task.
type GoogleTaskView struct {
	ID      string
	Title   string
	Notes   string
	Status  string
	Due     string
	Updated string
}

// TickTickTaskView is an application-level read projection of a created TickTick task.
type TickTickTaskView struct {
	ID string
}
