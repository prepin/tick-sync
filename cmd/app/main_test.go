package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
)

// runSync logs "sync started" and "sync finished" when RunOnce succeeds.
func TestRunSyncLogsStartedAndFinishedOnSuccess(t *testing.T) {
	ctx := context.Background()
	buf := captureLogOutput(t)

	runSync(ctx, &stubRunner{})

	if !strings.Contains(buf.String(), "sync started") {
		t.Fatalf("expected sync started log, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "sync finished") {
		t.Fatalf("expected sync finished log, got %q", buf.String())
	}
}

// runSync logs an error message without "sync finished" when RunOnce fails.
func TestRunSyncLogsErrorWhenRunOnceFails(t *testing.T) {
	ctx := context.Background()
	buf := captureLogOutput(t)

	runSync(ctx, &stubRunner{err: errors.New("ticktick unavailable")})

	if !strings.Contains(buf.String(), "sync started") {
		t.Fatalf("expected sync started log, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "sync finished with error: ticktick unavailable") {
		t.Fatalf("expected error log, got %q", buf.String())
	}
	if strings.Contains(buf.String(), "sync finished\n") {
		t.Fatal("did not expect sync finished log after error")
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
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(prev)
	})

	return &buf
}
