package httpserver

import (
	"net"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/config"
)

// Returns an error when the configured HTTP address cannot be bound.
func TestServerRunReturnsListenError(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	server := New(config.Config{HTTPAddr: listener.Addr().String()}, newTestTokenRepo(t))
	err = server.Run(t.Context())
	if err == nil {
		t.Fatal("expected listen error")
	}
	if !strings.Contains(err.Error(), "serve http") {
		t.Fatalf("unexpected error: %v", err)
	}
}
