package main

import "runtime/debug"

// revision is the commit the binary was built from, when the build says so:
// the Dockerfile passes it with -ldflags "-X main.revision=<commit>", because
// an image is built from a checkout with no .git for Go to read.
var revision string

// version is what `turnia version` prints: which commit is running.
//
// stamped is the revision set at build time, and wins when it isn't empty.
// Otherwise the answer comes from info, what Go recorded about the build
// (debug.ReadBuildInfo; nil when there's none). A build from a git checkout
// records the commit in the setting "vcs.revision", and "vcs.modified" is
// "true" if files had uncommitted changes, which adds "-dirty". With neither,
// it's "unknown".
func version(stamped string, info *debug.BuildInfo) string {
	if stamped != "" {
		return stamped
	}

	if info == nil {
		return "unknown"
	}

	var rev string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	if rev == "" {
		return "unknown"
	}

	if dirty {
		return rev + "-dirty"
	}

	return rev
}

// currentVersion is version for this binary.
func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	return version(revision, info)
}
