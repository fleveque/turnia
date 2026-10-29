// Package config reads the server's settings from environment variables.
//
// Every setting has a default that works for local development, so
// `turnia serve` runs with no environment at all. Production overrides them
// through Kamal's env section.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Config is everything the server needs to start.
type Config struct {
	// Addr is the address the HTTP server listens on, such as ":8080".
	Addr string
	// LogLevel is the lowest level that gets logged.
	LogLevel slog.Level
	// LogFormat is "json" (production) or "text" (easier to read locally).
	LogFormat string
	// ShutdownTimeout is how long in-flight requests get to finish after a
	// stop signal before the server closes their connections.
	ShutdownTimeout time.Duration
	// DatabaseURL is the Postgres connection URL the server uses. It connects
	// as turnia_app, which reads and writes rows but can't change the schema
	// (ADR 0002). The default is the development database from
	// docker-compose.yml; production sets it.
	DatabaseURL string
	// MigrateDatabaseURL is the connection URL `turnia migrate` uses. It
	// connects as the tables' owner, the one role that may change the schema.
	MigrateDatabaseURL string
}

// Load builds a Config from getenv, which is os.Getenv in production and a
// map lookup in tests. It reports every invalid variable at once, not just
// the first.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Addr:               ":8080",
		LogLevel:           slog.LevelInfo,
		LogFormat:          "json",
		ShutdownTimeout:    10 * time.Second,
		DatabaseURL:        "postgres://turnia_app:turnia_app@localhost:5433/turnia?sslmode=disable",
		MigrateDatabaseURL: "postgres://turnia:turnia@localhost:5433/turnia?sslmode=disable",
	}
	var errs []error

	if v := getenv("TURNIA_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := getenv("TURNIA_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := getenv("TURNIA_MIGRATE_DATABASE_URL"); v != "" {
		cfg.MigrateDatabaseURL = v
	}
	if v := getenv("TURNIA_LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			errs = append(errs, fmt.Errorf("TURNIA_LOG_LEVEL: %w", err))
		}
	}
	if v := getenv("TURNIA_LOG_FORMAT"); v != "" {
		if v != "json" && v != "text" {
			errs = append(errs, fmt.Errorf("TURNIA_LOG_FORMAT: %q is not json or text", v))
		}
		cfg.LogFormat = v
	}
	if v := getenv("TURNIA_SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("TURNIA_SHUTDOWN_TIMEOUT: %w", err))
		}
		cfg.ShutdownTimeout = d
	}

	return cfg, errors.Join(errs...)
}
