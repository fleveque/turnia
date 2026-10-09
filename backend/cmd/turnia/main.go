// Command turnia is the whole backend: one binary with subcommands.
//
//	turnia serve    run the HTTP server
//
// migrate and seed arrive with the database (milestones 2 and 13).
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/fleveque/turnia/backend/internal/config"
	"github.com/fleveque/turnia/backend/internal/server"
)

const usage = `usage: turnia <command>

commands:
  serve    run the HTTP server
`

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

// run is main without the process: everything it needs comes in as
// arguments, so tests can call it, and it returns the exit code instead of
// calling os.Exit, so deferred calls run.
func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "serve":
		return serve(getenv, stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "turnia: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func serve(getenv func(string) string, stdout, stderr io.Writer) int {
	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintf(stderr, "turnia: invalid configuration:\n%v\n", err)
		return 1
	}
	log := newLogger(cfg, stdout)
	slog.SetDefault(log)

	// Ctrl-C locally, SIGTERM from Docker when Kamal replaces the container.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Error("listening", "addr", cfg.Addr, "err", err)
		return 1
	}
	log.Info("listening", "addr", ln.Addr().String())

	srv := server.New(server.Handler(log), log)
	if err := server.Run(ctx, srv, ln, cfg.ShutdownTimeout); err != nil {
		log.Error("server stopped", "err", err)
		return 1
	}
	log.Info("stopped")
	return 0
}

func newLogger(cfg config.Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "text" {
		return slog.New(slog.NewTextHandler(w, opts))
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}
