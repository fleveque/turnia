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
	// YOUR TURN (milestone 4.1): the body, in four steps.
	//
	//  1. If stamped isn't "", return it.
	//  2. If info is nil (no build info at all), return "unknown".
	//  3. info.Settings is a slice of debug.BuildSetting, a struct with two
	//     string fields, Key and Value. Go through it with
	//     `for _, s := range info.Settings { … }` and remember two things:
	//     the Value of the setting whose Key is "vcs.revision", and whether
	//     the setting whose Key is "vcs.modified" has the Value "true".
	//     (Other settings, like "GOARCH", are there too; skip them.)
	//  4. If no revision was found, return "unknown". If there was one and
	//     the files were modified, return the revision followed by "-dirty".
	//     Otherwise, return the revision.
	//
	// The tests in version_test.go say what's expected; from backend/,
	// `go test ./cmd/turnia/` runs them.
	return ""
}

// currentVersion is version for this binary.
func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	return version(revision, info)
}
