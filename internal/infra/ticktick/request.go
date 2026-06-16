package ticktick

// createTaskRequest is the exact JSON shape expected by the TickTick API.
type createTaskRequest struct {
	Title     string `json:"title"`
	ProjectID string `json:"projectId,omitempty"`
	Content   string `json:"content,omitempty"`
	DueDate   string `json:"dueDate,omitempty"`
	TimeZone  string `json:"timeZone,omitempty"`
	IsAllDay  *bool  `json:"isAllDay,omitempty"`
}
