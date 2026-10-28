package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// env turns a map into a getenv function.
func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{
		Addr:               ":8080",
		LogLevel:           slog.LevelInfo,
		LogFormat:          "json",
		ShutdownTimeout:    10 * time.Second,
		DatabaseURL:        "postgres://turnia_app:turnia_app@localhost:5433/turnia?sslmode=disable",
		MigrateDatabaseURL: "postgres://turnia:turnia@localhost:5433/turnia?sslmode=disable",
	}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"TURNIA_ADDR":                 "127.0.0.1:9000",
		"TURNIA_LOG_LEVEL":            "debug",
		"TURNIA_LOG_FORMAT":           "text",
		"TURNIA_SHUTDOWN_TIMEOUT":     "3s",
		"TURNIA_DATABASE_URL":         "postgres://app@prod/turnia",
		"TURNIA_MIGRATE_DATABASE_URL": "postgres://owner@prod/turnia",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{
		Addr:               "127.0.0.1:9000",
		LogLevel:           slog.LevelDebug,
		LogFormat:          "text",
		ShutdownTimeout:    3 * time.Second,
		DatabaseURL:        "postgres://app@prod/turnia",
		MigrateDatabaseURL: "postgres://owner@prod/turnia",
	}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

// Two settings, not one URL used for both: the server must never run as the
// owner because the migrate URL was copied into it, or the other way round.
func TestLoadKeepsTheTwoDatabaseURLsApart(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"TURNIA_DATABASE_URL": "postgres://app@prod/turnia",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := "postgres://turnia:turnia@localhost:5433/turnia?sslmode=disable"; cfg.MigrateDatabaseURL != want {
		t.Errorf("MigrateDatabaseURL = %q, want the default %q", cfg.MigrateDatabaseURL, want)
	}

	cfg, err = Load(env(map[string]string{
		"TURNIA_MIGRATE_DATABASE_URL": "postgres://owner@prod/turnia",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := "postgres://turnia_app:turnia_app@localhost:5433/turnia?sslmode=disable"; cfg.DatabaseURL != want {
		t.Errorf("DatabaseURL = %q, want the default %q", cfg.DatabaseURL, want)
	}
}

func TestLoadReportsEveryInvalidVariable(t *testing.T) {
	_, err := Load(env(map[string]string{
		"TURNIA_LOG_LEVEL":        "loud",
		"TURNIA_LOG_FORMAT":       "xml",
		"TURNIA_SHUTDOWN_TIMEOUT": "soon",
	}))
	if err == nil {
		t.Fatal("Load accepted invalid variables")
	}
	for _, name := range []string{"TURNIA_LOG_LEVEL", "TURNIA_LOG_FORMAT", "TURNIA_SHUTDOWN_TIMEOUT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error doesn't mention %s: %v", name, err)
		}
	}
}
