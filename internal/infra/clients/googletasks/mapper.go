package googletasks

import (
	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
	tasksapi "google.golang.org/api/tasks/v1"
)

func toGoogleTaskView(task *tasksapi.Task) googletasksync.GoogleTaskView {
	if task == nil {
		return googletasksync.GoogleTaskView{}
	}

	return googletasksync.GoogleTaskView{
		ID:      task.Id,
		Title:   task.Title,
		Notes:   task.Notes,
		Status:  task.Status,
		Due:     task.Due,
		Updated: task.Updated,
	}
}
