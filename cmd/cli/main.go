package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	googleclient "github.com/prepin/tick-sync/internal/clients/google"
	ticktickclient "github.com/prepin/tick-sync/internal/clients/ticktick"
	"github.com/prepin/tick-sync/internal/config"
	sqlitestore "github.com/prepin/tick-sync/internal/infra/sqlite"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
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
	postSyncAction, err := postSyncActionFromConfig(cfg.GooglePostSyncAction)
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite db: %w", err)
	}
	defer db.Close()

	store, err := sqlitestore.NewGoogleTasksStore(ctx, db)
	if err != nil {
		return fmt.Errorf("create google tasks store: %w", err)
	}

	google, err := googleclient.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create google tasks client: %w", err)
	}

	ticktick, err := ticktickclient.New(cfg)
	if err != nil {
		return fmt.Errorf("create ticktick client: %w", err)
	}

	uc := googletasksync.New(google, ticktick, store, postSyncAction)
	summary, syncErr := uc.SyncGoogleTasksToTickTick(ctx)
	printSyncSummary(out, summary)
	if syncErr != nil {
		return fmt.Errorf("sync google tasks to ticktick: %w", syncErr)
	}

	return nil
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

func postSyncActionFromConfig(value string) (googletasksync.PostSyncAction, error) {
	switch strings.TrimSpace(value) {
	case "", "complete":
		return googletasksync.PostSyncActionComplete, nil
	case "delete":
		return googletasksync.PostSyncActionDelete, nil
	default:
		return "", fmt.Errorf("unsupported GOOGLE_POST_SYNC_ACTION %q; expected complete or delete", value)
	}
}

func printSyncSummary(out io.Writer, summary googletasksync.SyncSummary) {
	fmt.Fprintln(out, "Sync summary:")
	fmt.Fprintf(out, "Seen: %d\n", summary.Seen)
	fmt.Fprintf(out, "Created: %d\n", summary.Created)
	fmt.Fprintf(out, "Skipped: %d\n", summary.Skipped)
	fmt.Fprintf(out, "Failed: %d\n", summary.Failed)
	fmt.Fprintf(out, "Completed: %d\n", summary.Completed)
	fmt.Fprintf(out, "Deleted: %d\n", summary.Deleted)

	if len(summary.Errors) == 0 {
		return
	}

	fmt.Fprintln(out, "Errors:")
	for _, err := range summary.Errors {
		if err == nil {
			continue
		}
		fmt.Fprintf(out, "- %v\n", err)
	}
}
