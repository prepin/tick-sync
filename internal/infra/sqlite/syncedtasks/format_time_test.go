package syncedtasks

import (
	"testing"
	"time"
)

// Converts a time value to RFC3339Nano format in UTC regardless of the input timezone.
func TestFormatTimeReturnsRFC3339NanoUTC(t *testing.T) {
	t.Parallel()
	input := time.Date(2026, 6, 10, 12, 0, 0, 123, time.FixedZone("EST", -5*60*60))
	got := formatTime(input)
	if got != "2026-06-10T17:00:00.000000123Z" {
		t.Fatalf("unexpected formatted time: %q", got)
	}
}
