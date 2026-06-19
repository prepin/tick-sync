// Package httpserver provides the local web UI and OAuth callbacks.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
)

// Server runs the local web UI.
type Server struct {
	server             *http.Server
	logger             *slog.Logger
	basicAuthEnabled   bool
	publicBindPossible bool
}

// New creates a local HTTP server for browser-based auth flows.
func New(cfg config.Config, tokens TokenStore, opts ...Option) *Server {
	h := newHandler(cfg, tokens)
	s := &Server{
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           h.routes(),
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger:             slog.New(slog.DiscardHandler),
		basicAuthEnabled:   cfg.HTTPBasicAuthPassword != "",
		publicBindPossible: isPublicBindAddress(cfg.HTTPAddr),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Option configures the HTTP server.
type Option func(*Server)

// WithLogger configures the logger used by the HTTP server.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// Start begins serving until the context is cancelled.
func (s *Server) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Server) run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.WarnContext(ctx, "http server shutdown failed", "error", err)
		}
	}()

	s.logger.InfoContext(ctx, "http server started", "addr", s.server.Addr)
	if s.publicBindPossible && !s.basicAuthEnabled {
		s.logger.WarnContext(
			ctx,
			"http server may be reachable from other hosts without basic auth",
			"addr",
			s.server.Addr,
		)
	}
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.ErrorContext(ctx, "http server failed", "error", err)
	}
}

func isPublicBindAddress(addr string) bool {
	if strings.HasPrefix(addr, ":") {
		return true
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		return true
	}

	parsed := net.ParseIP(strings.Trim(host, "[]"))
	return parsed != nil && parsed.IsUnspecified()
}

// TokenStore is the storage required by the auth handlers.
type TokenStore interface {
	Get(ctx context.Context, provider string) (oauthtokens.Token, error)
	Save(ctx context.Context, provider string, token oauthtokens.Token) error
}

func statusText(ctx context.Context, tokens TokenStore, provider string, name string) string {
	token, err := tokens.Get(ctx, provider)
	if err == nil {
		if token.ExpiresAt.IsZero() {
			return name + " is connected."
		}
		return fmt.Sprintf("%s is connected. Token expires at %s.", name, token.ExpiresAt.Format(time.RFC3339))
	}
	if errors.Is(err, oauthtokens.ErrTokenNotFound) {
		return name + " is not connected."
	}
	return name + " token status is unavailable."
}
