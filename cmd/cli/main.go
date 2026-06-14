package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	googleclient "github.com/prepin/tick-sync/internal/clients/google"
	ticktickclient "github.com/prepin/tick-sync/internal/clients/ticktick"
	"github.com/prepin/tick-sync/internal/config"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
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
		runErr = runSync(ctx, cfg)
	default:
		runErr = fmt.Errorf("unknown command %q; expected list or sync", command)
	}
	if runErr != nil {
		slog.Error(runErr.Error())
		os.Exit(1)
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

func runSync(ctx context.Context, cfg config.Config) error {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite db: %w", err)
	}
	defer db.Close()

	repo, err := googletasksrepo.NewGoogleTasksRepo(ctx, db)
	if err != nil {
		return fmt.Errorf("create google tasks repo: %w", err)
	}

	google, err := googleclient.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create google tasks client: %w", err)
	}

	ticktick, err := ticktickclient.New(cfg)
	if err != nil {
		return fmt.Errorf("create ticktick client: %w", err)
	}

	uc := googletasksync.New(google, ticktick, repo, cfg.GooglePostSyncAction)

	summary, syncErr := uc.SyncGoogleTasksToTickTick(ctx)
	logSyncSummary(summary)
	if syncErr != nil {
		return fmt.Errorf("sync google tasks to ticktick: %w", syncErr)
	}

	return nil
}

func logSyncSummary(summary googletasksync.SyncSummary) {
	attrs := []slog.Attr{
		slog.Int("seen", summary.Seen),
		slog.Int("created", summary.Created),
		slog.Int("skipped", summary.Skipped),
		slog.Int("failed", summary.Failed),
		slog.Int("completed", summary.Completed),
		slog.Int("deleted", summary.Deleted),
	}

	if len(summary.Errors) > 0 {
		nonNil := make([]string, 0, len(summary.Errors))
		for _, err := range summary.Errors {
			if err != nil {
				nonNil = append(nonNil, err.Error())
			}
		}
		if len(nonNil) > 0 {
			attrs = append(attrs, slog.String("errors", strings.Join(nonNil, ", ")))
		}
	}

	if summary.Failed > 0 || len(summary.Errors) > 0 {
		slog.LogAttrs(context.Background(), slog.LevelError, "sync finished", attrs...)
	} else {
		slog.LogAttrs(context.Background(), slog.LevelInfo, "sync finished", attrs...)
	}
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
