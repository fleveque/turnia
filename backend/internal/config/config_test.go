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
	want := Config{Addr: ":8080", LogLevel: slog.LevelInfo, LogFormat: "json", ShutdownTimeout: 10 * time.Second}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"TURNIA_ADDR":             "127.0.0.1:9000",
		"TURNIA_LOG_LEVEL":        "debug",
		"TURNIA_LOG_FORMAT":       "text",
		"TURNIA_SHUTDOWN_TIMEOUT": "3s",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{Addr: "127.0.0.1:9000", LogLevel: slog.LevelDebug, LogFormat: "text", ShutdownTimeout: 3 * time.Second}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
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
