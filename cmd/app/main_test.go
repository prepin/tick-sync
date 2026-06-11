package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// runSync logs "sync started" and returns nil when RunOnce succeeds.
func TestRunSyncLogsStartedAndReturnsNilOnSuccess(t *testing.T) {
	ctx := context.Background()
	buf := captureLogOutput(t)

	if err := runSync(ctx, &stubRunner{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `msg="sync started"`) {
		t.Fatalf("expected sync started log, got %q", got)
	}
}

// runSync returns the error from RunOnce and does not log "sync finished".
func TestRunSyncReturnsErrorWhenRunOnceFails(t *testing.T) {
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
	if strings.Contains(got, `msg="sync finished"`) {
		t.Fatal("did not expect sync finished from runSync; PrintSyncSummary logs it inside RunOnce")
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
