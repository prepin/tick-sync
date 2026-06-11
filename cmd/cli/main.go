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
	"github.com/prepin/tick-sync/internal/service"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	command := "list"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	var runErr error
	switch command {
	case "list":
		runErr = runList(ctx, cfg, os.Stdout)
	case "sync":
		runErr = runSync(ctx, cfg, os.Stdout)
	default:
		runErr = fmt.Errorf("unknown command %q; expected list or sync", command)
	}
	if runErr != nil {
		log.Fatal(runErr)
	}
}

func runList(ctx context.Context, cfg config.Config, out io.Writer) error {
	client, err := googleclient.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create google tasks client: %w", err)
	}

	tasks, err := client.ListUncompleted(ctx)
	if err != nil {
		return fmt.Errorf("list google tasks: %w", err)
	}

	printTasks(out, cfg.GoogleTaskListID, tasks)
	return nil
}

func runSync(ctx context.Context, cfg config.Config, out io.Writer) error {
	runner, cleanup, err := service.NewSyncRunner(ctx, cfg, out)
	if err != nil {
		return err
	}
	defer cleanup()

	return runner.RunOnce(ctx)
}

func printTasks(out io.Writer, taskListID string, tasks []googletasksync.GoogleTask) {
	fmt.Fprintf(out, "Google task list: %s\n", taskListID)

	if len(tasks) == 0 {
		fmt.Fprintln(out, "No uncompleted tasks found.")
		return
	}

	for _, task := range tasks {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "- id: %s\n", task.ID)
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
		for line := range strings.SplitSeq(notes, "\n") {
			fmt.Fprintf(out, "    %s\n", line)
		}
	}
}
