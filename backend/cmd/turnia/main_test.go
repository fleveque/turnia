package main

import (
	"bytes"
	"strings"
	"testing"
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
