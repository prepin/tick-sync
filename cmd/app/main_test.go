package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// runSync logs "sync started" and "sync finished" when RunOnce succeeds.
func TestRunSyncLogsStartedAndFinishedOnSuccess(t *testing.T) {
	ctx := context.Background()
	buf := captureLogOutput(t)

	if err := runSync(ctx, &stubRunner{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `msg="sync started"`) {
		t.Fatalf("expected sync started log, got %q", got)
	}
	if !strings.Contains(got, `msg="sync finished"`) {
		t.Fatalf("expected sync finished log, got %q", got)
	}
}

// runSync returns the error from RunOnce and logs it without a sync finished info line.
func TestRunSyncLogsErrorWhenRunOnceFails(t *testing.T) {
	ctx := context.Background()
	buf := captureLogOutput(t)

	gotErr := runSync(ctx, &stubRunner{err: errors.New("ticktick unavailable")})
	if gotErr == nil {
		t.Fatal("expected error")
	}
	if gotErr.Error() != "ticktick unavailable" {
		t.Fatalf("unexpected error: %v", gotErr)
	}

	got := buf.String()
	if !strings.Contains(got, `msg="sync started"`) {
		t.Fatalf("expected sync started log, got %q", got)
	}
	if !strings.Contains(got, `level=ERROR`) {
		t.Fatalf("expected error level log, got %q", got)
	}
	if !strings.Contains(got, `error="ticktick unavailable"`) {
		t.Fatalf("expected error detail in log, got %q", got)
	}

	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "sync finished") && strings.Contains(line, "level=INFO") {
			t.Fatalf("did not expect sync finished info log after error, got line %q", line)
		}
	}
}

type stubRunner struct {
	err error
}

func (r *stubRunner) RunOnce(_ context.Context) error {
	return r.err
}

func captureLogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return &buf
}
