package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fleveque/turnia/backend/internal/platform/db/dbtest"
)

func TestRunWithoutACommandPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, func(string) string { return "" }, &stdout, &stderr); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage: turnia") {
		t.Errorf("stderr = %q, want usage", stderr.String())
	}
}

func TestRunRejectsAnUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"deploy"}, func(string) string { return "" }, &stdout, &stderr); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unknown command "deploy"`) {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestServeRefusesInvalidConfiguration(t *testing.T) {
	var stdout, stderr bytes.Buffer
	getenv := func(k string) string {
		if k == "TURNIA_LOG_FORMAT" {
			return "xml"
		}
		return ""
	}
	if code := run([]string{"serve"}, getenv, &stdout, &stderr); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "TURNIA_LOG_FORMAT") {
		t.Errorf("stderr = %q, want it to name the variable", stderr.String())
	}
}

// Tests for migrate, written before it: they're the specification.

func TestMigrateRefusesInvalidConfiguration(t *testing.T) {
	var stdout, stderr bytes.Buffer
	getenv := func(k string) string {
		if k == "TURNIA_LOG_FORMAT" {
			return "xml"
		}
		return ""
	}
	if code := run([]string{"migrate"}, getenv, &stdout, &stderr); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid configuration") || !strings.Contains(stderr.String(), "TURNIA_LOG_FORMAT") {
		t.Errorf("stderr = %q, want the invalid configuration message naming the variable", stderr.String())
	}
}

func TestMigrateAppliesTheMigrationsOnce(t *testing.T) {
	url := dbtest.NewDatabase(t)
	getenv := func(k string) string {
		if k == "TURNIA_DATABASE_URL" {
			return url
		}
		return ""
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"migrate"}, getenv, &stdout, &stderr); code != 0 {
		t.Fatalf("first run: exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"msg":"migrated"`) {
		t.Errorf("first run should log each migration; stdout:\n%s", stdout.String())
	}

	stdout.Reset()
	if code := run([]string{"migrate"}, getenv, &stdout, &stderr); code != 0 {
		t.Fatalf("second run: exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "database up to date") {
		t.Errorf("second run should find nothing to do; stdout:\n%s", stdout.String())
	}
}

func TestMigrateFailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	getenv := func(k string) string {
		if k == "TURNIA_DATABASE_URL" {
			// Port 1: nothing listens there, so the connection is refused at once.
			return "postgres://turnia:turnia@127.0.0.1:1/turnia?sslmode=disable&connect_timeout=2"
		}
		return ""
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"migrate"}, getenv, &stdout, &stderr); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"level":"ERROR"`) {
		t.Errorf("want the error in the log (stdout); got:\n%s", stdout.String())
	}
}
