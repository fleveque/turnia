package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

// built returns build info with these settings, as Go records them.
func built(settings ...debug.BuildSetting) *debug.BuildInfo {
	return &debug.BuildInfo{Settings: settings}
}

func TestVersion(t *testing.T) {
	const commit = "3660747eed81e1745782a669fd16c10a18e552f1"
	tests := []struct {
		name    string
		stamped string
		info    *debug.BuildInfo
		want    string
	}{
		{"stamped at build time", commit, nil, commit},
		{"stamped wins over Go's record", commit, built(debug.BuildSetting{Key: "vcs.revision", Value: "other"}), commit},
		{"from Go's record", "", built(
			debug.BuildSetting{Key: "GOARCH", Value: "arm64"},
			debug.BuildSetting{Key: "vcs.revision", Value: commit},
			debug.BuildSetting{Key: "vcs.modified", Value: "false"},
		), commit},
		{"uncommitted changes", "", built(
			debug.BuildSetting{Key: "vcs.revision", Value: commit},
			debug.BuildSetting{Key: "vcs.modified", Value: "true"},
		), commit + "-dirty"},
		{"a record without a commit", "", built(debug.BuildSetting{Key: "GOARCH", Value: "arm64"}), "unknown"},
		{"no record at all", "", nil, "unknown"},
	}
	for _, tt := range tests {
		if got := version(tt.stamped, tt.info); got != tt.want {
			t.Errorf("%s: version = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRunVersionPrintsOneLine(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"version"}, func(string) string { return "" }, &stdout, &stderr); code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr %q)", code, stderr.String())
	}
	if out := stdout.String(); out == "" || strings.Count(out, "\n") != 1 {
		t.Errorf("stdout = %q, want one line", out)
	}
}
