// Package tickticktokenreminder provides a cron job that reminds about expiring TickTick tokens.
package tickticktokenreminder

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder"
)

// Job polls the reminder use case on a configured interval.
type Job struct {
	usecase  *tickticktokenreminder.UseCase
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

// New creates a token reminder job.
func New(usecase *tickticktokenreminder.UseCase, interval time.Duration, opts ...JobOption) *Job {
	j := &Job{usecase: usecase, interval: interval, logger: slog.New(slog.DiscardHandler)}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// Name returns the job identifier.
func (j *Job) Name() string {
	return "ticktick-token-reminder"
}

// Run starts the polling loop until the context is cancelled.
func (j *Job) Run(ctx context.Context) error {
	j.logger.InfoContext(ctx, "job started", "job", j.Name())
	if err := j.Execute(ctx); err != nil {
		j.logger.ErrorContext(ctx, "job initial reminder check failed", "job", j.Name(), "error", err)
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			j.logger.InfoContext(ctx, "job shutting down", "job", j.Name())
			return nil
		case <-ticker.C:
			if err := j.Execute(ctx); err != nil {
				j.logger.ErrorContext(ctx, "job reminder check failed", "job", j.Name(), "error", err)
			}
		}
	}
}

// Execute runs one reminder check.
func (j *Job) Execute(ctx context.Context) error {
	if err := j.usecase.Handle(ctx); err != nil {
		return fmt.Errorf("check ticktick token refresh reminder: %w", err)
	}
	return nil
}
