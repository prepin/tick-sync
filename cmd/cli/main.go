package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	googleclient "github.com/prepin/tick-sync/internal/clients/google"
	"github.com/prepin/tick-sync/internal/config"
	googletasks "google.golang.org/api/tasks/v1"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	client, err := googleclient.New(ctx, cfg)
	if err != nil {
		log.Fatalf("create google tasks client: %v", err)
	}

	tasks, err := client.ListUncompletedTasks(ctx, cfg.GoogleTaskListID)
	if err != nil {
		log.Fatalf("list google tasks: %v", err)
	}

	printTasks(os.Stdout, cfg.GoogleTaskListID, tasks)
}

func printTasks(out io.Writer, taskListID string, tasks []*googletasks.Task) {
	fmt.Fprintf(out, "Google task list: %s\n", taskListID)

	if len(tasks) == 0 {
		fmt.Fprintln(out, "No uncompleted tasks found.")
		return
	}

	for _, task := range tasks {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "- id: %s\n", task.Id)
		fmt.Fprintf(out, "  title: %s\n", task.Title)
		fmt.Fprintf(out, "  status: %s\n", task.Status)
		fmt.Fprintf(out, "  due: %s\n", task.Due)
		fmt.Fprintf(out, "  updated: %s\n", task.Updated)

		notes := strings.TrimSpace(task.Notes)
		if notes == "" {
			fmt.Fprintln(out, "  notes:")
			continue
		}

		fmt.Fprintln(out, "  notes: |")
		for _, line := range strings.Split(notes, "\n") {
			fmt.Fprintf(out, "    %s\n", line)
		}
	}
}
