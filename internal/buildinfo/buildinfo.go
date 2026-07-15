// Package buildinfo carries the version metadata stamped into the binary at
// build time via -ldflags (see the Makefile and .goreleaser.yaml).
package buildinfo

import "fmt"

var (
	// Version is the release version (e.g. "6.0.0"), or "dev" for local builds.
	Version = "dev"
	// Commit is the short git commit the binary was built from.
	Commit = "none"
	// Date is the UTC build timestamp.
	Date = "unknown"
)

// String renders the one-line version report used by both `gayle --version`
// and release smoke checks.
func String() string {
	return fmt.Sprintf("gayle %s (commit %s, built %s)", Version, Commit, Date)
}
