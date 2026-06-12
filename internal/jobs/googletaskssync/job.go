package googletaskssyncjob

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

type Job struct {
	usecase  *googletasksync.Usecase
	interval time.Duration
}

func New(usecase *googletasksync.Usecase, interval time.Duration) *Job {
	return &Job{usecase: usecase, interval: interval}
}

func (j *Job) Name() string {
	return "google-tasks-sync"
}

func (j *Job) Start(ctx context.Context) {
	go j.run(ctx)
}

func (j *Job) run(ctx context.Context) {
	slog.Info("job started", "job", j.Name())

	if err := j.Execute(ctx); err != nil {
		slog.Error("job initial sync failed", "job", j.Name(), "error", err)
		return
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("job shutting down", "job", j.Name())
			return
		case <-ticker.C:
			if err := j.Execute(ctx); err != nil {
				slog.Error("job sync failed", "job", j.Name(), "error", err)
			}
		}
	}
}

func (j *Job) Execute(ctx context.Context) error {
	summary, syncErr := j.usecase.SyncGoogleTasksToTickTick(ctx)
	j.logSyncSummary(summary)
	if syncErr != nil {
		return fmt.Errorf("sync google tasks to ticktick: %w", syncErr)
	}
	return nil
}

func (j *Job) logSyncSummary(summary googletasksync.SyncSummary) {
	attrs := []slog.Attr{
		slog.String("job", j.Name()),
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
