// Package googletasksync provides a cron job that syncs Google Tasks to TickTick.
package googletasksync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// Job polls the sync use case on a configured interval.
type Job struct {
	usecase  *googletasksync.SyncGoogleTasksToTickTickUseCase
	interval time.Duration
	logger   *slog.Logger
}

// JobOption configures a Job.
type JobOption func(*Job)

// WithLogger configures the logger used by the job.
func WithLogger(logger *slog.Logger) JobOption {
	return func(j *Job) {
		j.logger = logger
	}
}

// New creates a sync job.
func New(usecase *googletasksync.SyncGoogleTasksToTickTickUseCase, interval time.Duration, opts ...JobOption) *Job {
	j := &Job{
		usecase:  usecase,
		interval: interval,
		logger:   slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// Name returns the job identifier.
func (j *Job) Name() string {
	return "google-tasks-sync"
}

// Start begins the polling loop.
func (j *Job) Start(ctx context.Context) {
	go j.run(ctx)
}

func (j *Job) run(ctx context.Context) {
	j.logger.InfoContext(ctx, "job started", "job", j.Name())

	if err := j.Execute(ctx); err != nil {
		j.logger.ErrorContext(ctx, "job initial sync failed", "job", j.Name(), "error", err)
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.InfoContext(ctx, "job shutting down", "job", j.Name())
			return
		case <-ticker.C:
			if err := j.Execute(ctx); err != nil {
				j.logger.ErrorContext(ctx, "job sync failed", "job", j.Name(), "error", err)
			}
		}
	}
}

// Execute runs one sync cycle.
func (j *Job) Execute(ctx context.Context) error {
	result, syncErr := j.usecase.Handle(ctx)
	j.logSyncResult(ctx, result)
	if syncErr != nil {
		return fmt.Errorf("sync google tasks to ticktick: %w", syncErr)
	}
	return nil
}

func (j *Job) logSyncResult(ctx context.Context, result googletasksync.SyncGoogleTasksToTickTickResult) {
	attrs := []slog.Attr{
		slog.String("job", j.Name()),
		slog.Int("seen", result.Seen),
		slog.Int("created", result.Created),
		slog.Int("skipped", result.Skipped),
		slog.Int("delayed", result.Delayed),
		slog.Int("failed", result.Failed),
		slog.Int("completed", result.Completed),
		slog.Int("deleted", result.Deleted),
	}

	if len(result.Errors) > 0 {
		nonNil := make([]string, 0, len(result.Errors))
		for _, err := range result.Errors {
			if err != nil {
				nonNil = append(nonNil, err.Error())
			}
		}
		if len(nonNil) > 0 {
			attrs = append(attrs, slog.String("errors", strings.Join(nonNil, ", ")))
		}
	}

	if result.Failed > 0 || len(result.Errors) > 0 {
		j.logger.LogAttrs(ctx, slog.LevelError, "sync finished", attrs...)
	} else {
		j.logger.LogAttrs(ctx, slog.LevelInfo, "sync finished", attrs...)
	}
}
