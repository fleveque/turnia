// Package server wires the HTTP routes and middleware together, and runs the
// server until it's told to stop.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/fleveque/turnia/backend/internal/platform/httpx"
)

// Pinger is what the health check needs from the database: *pgxpool.Pool
// has this method, and tests pass a fake that has it too.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler returns the whole API: every route, wrapped in the middleware
// every request goes through.
func Handler(log *slog.Logger, db Pinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/healthz", healthz(db, log))

	// Anything else under /api is a JSON 404, so API clients always get a
	// Problem. (The rest of the paths will serve the PWA, from milestone 14.)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, http.StatusNotFound, "no route for "+r.Method+" "+r.URL.Path)
	})

	// Logger is outside Recover, so a panic reaches the log as a 500.
	return httpx.Logger(log)(httpx.Recover(log)(mux))
}

// healthTimeout is how long the health check waits for the database. A
// database that takes longer than this to answer "SELECT 1"-sized questions
// is in trouble even if it isn't down.
const healthTimeout = 2 * time.Second

// healthz reports whether the server can reach its database: 200 with
// {"status":"ok"} if db answers within healthTimeout, 503 with
// {"status":"unavailable"} if it doesn't. The error goes to the log, never
// to the response, which anyone can read.
func healthz(db Pinger, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			log.Warn("health check: database", "err", err)
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}

		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// New returns an http.Server for h with timeouts set. The zero http.Server
// has none: a client that opens a connection and sends nothing holds it
// forever.
func New(h http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		// net/http's own errors (TLS handshakes, bad requests) go to the
		// same structured log as everything else.
		ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}

// Run serves on ln until ctx is cancelled, then shuts down gracefully: it
// stops accepting connections and waits up to shutdownTimeout for requests
// in flight to finish. It returns nil after a clean shutdown.
func Run(ctx context.Context, srv *http.Server, ln net.Listener, shutdownTimeout time.Duration) error {
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		// Serve returned before we asked it to stop.
		return fmt.Errorf("serving: %w", err)
	case <-ctx.Done():
	}

	// ctx is already cancelled, so the shutdown deadline needs a context
	// of its own. WithoutCancel keeps ctx's values but not its cancellation.
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serving: %w", err)
	}
	return nil
}
