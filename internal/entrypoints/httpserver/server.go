// Package httpserver provides the local web UI and OAuth callbacks.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/infra/sqlite/tickticktokens"
)

// Server runs the local web UI.
type Server struct {
	server *http.Server
	logger *slog.Logger
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
		logger: slog.New(slog.DiscardHandler),
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.WarnContext(ctx, "http server shutdown failed", "error", err)
		}
	}()

	s.logger.InfoContext(ctx, "http server started", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.ErrorContext(ctx, "http server failed", "error", err)
	}
}

// TokenStore is the storage required by the auth handlers.
type TokenStore interface {
	Get(ctx context.Context) (tickticktokens.Token, error)
	Save(ctx context.Context, token tickticktokens.Token) error
}

func statusText(ctx context.Context, tokens TokenStore) string {
	token, err := tokens.Get(ctx)
	if err == nil {
		if token.ExpiresAt.IsZero() {
			return "TickTick is connected."
		}
		return fmt.Sprintf("TickTick is connected. Token expires at %s.", token.ExpiresAt.Format(time.RFC3339))
	}
	if errors.Is(err, tickticktokens.ErrTokenNotFound) {
		return "TickTick is not connected."
	}
	return "TickTick token status is unavailable."
}
