package ui

import "os"

// The Node CLI's logger contract, preserved because pipelines grep for it:
// every diagnostic line goes to STDERR with the literal "Gayle: " prefix.
// Stdout is reserved for machine output (the `fetch` result). Colour styling
// wraps the message only, never the prefix — exactly like the chalk calls it
// replaces.

// Log writes "Gayle: <msg>" to stderr.
func Log(msg string) {
	Fprintln(os.Stderr, "Gayle: "+msg)
}

// Warn writes a yellow "Gayle: WARNING: <msg>" line to stderr.
func Warn(msg string) {
	Log(yellowStyle.Render("WARNING: " + msg))
}

// Gray styles a status line (paths, update reports).
func Gray(s string) string { return grayStyle.Render(s) }

// Cyan styles a banner/header line (mode banner, section headers).
func Cyan(s string) string { return cyanStyle.Render(s) }

// Green styles an affirmative line (the final "Done.").
func Green(s string) string { return greenStyle.Render(s) }

// Yellow styles a cautionary line (cleanup deletions).
func Yellow(s string) string { return yellowStyle.Render(s) }
