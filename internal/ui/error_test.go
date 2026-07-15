package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/driverforge/gayle/internal/clierr"
)

// A bytes.Buffer is not a terminal, so these tests exercise the plain
// (pipe/CI) output path — the grep-able stderr contract pipelines see.

func TestRenderUserErrorPlain(t *testing.T) {
	var buf bytes.Buffer
	RenderUserError(&buf, clierr.UserT("Missing stage", "Invalid options!! You must specify stage.", "pass -s <stage>"))
	got := buf.String()
	if !strings.Contains(got, "Gayle: ERROR: Invalid options!! You must specify stage.\n") {
		t.Errorf("missing ERROR line:\n%s", got)
	}
	if !strings.Contains(got, "Gayle: TIP: pass -s <stage>\n") {
		t.Errorf("missing TIP line:\n%s", got)
	}
}

func TestRenderUserErrorPlainWithoutHint(t *testing.T) {
	var buf bytes.Buffer
	RenderUserError(&buf, clierr.User("something expected", ""))
	got := buf.String()
	if got != "Gayle: ERROR: something expected\n" {
		t.Errorf("plain render = %q", got)
	}
}

func TestRenderUserErrorSilent(t *testing.T) {
	var buf bytes.Buffer
	RenderUserError(&buf, clierr.Silent())
	if buf.Len() != 0 {
		t.Errorf("Silent must render nothing, got %q", buf.String())
	}
}

func TestRenderUserErrorNil(t *testing.T) {
	var buf bytes.Buffer
	RenderUserError(&buf, nil)
	RenderCrash(&buf, nil)
	if buf.Len() != 0 {
		t.Errorf("nil errors must render nothing, got %q", buf.String())
	}
}

func TestRenderCrashPlain(t *testing.T) {
	var buf bytes.Buffer
	RenderCrash(&buf, errors.New("nil pointer dereference"))
	if got := buf.String(); got != "Gayle: ERROR: nil pointer dereference\n" {
		t.Errorf("crash render = %q", got)
	}
}

// A wrapped UserError still dissects to its friendly parts (errors.As sees
// through fmt.Errorf %w chains).
func TestDissect(t *testing.T) {
	title, msg, hint := dissect(clierr.UserT("Cleanup failed", "3 parameters failed", "check IAM"))
	if title != "Cleanup failed" || msg != "3 parameters failed" || hint != "check IAM" {
		t.Errorf("dissect = %q %q %q", title, msg, hint)
	}

	// Untitled UserError defaults to "Heads up".
	title, _, _ = dissect(clierr.User("plain expected", ""))
	if title != "Heads up" {
		t.Errorf("default title = %q", title)
	}

	// Non-UserError: message is the error text, no hint.
	title, msg, hint = dissect(errors.New("boom"))
	if title != "Heads up" || msg != "boom" || hint != "" {
		t.Errorf("plain dissect = %q %q %q", title, msg, hint)
	}
}

func TestIsTerminalWriter(t *testing.T) {
	if isTerminalWriter(&bytes.Buffer{}) {
		t.Errorf("a buffer must not be a terminal")
	}
}

// card is TTY-only in practice, but its layout is pure string manipulation —
// pin the frame so a lipgloss upgrade can't silently break the borders.
func TestCardLayout(t *testing.T) {
	got := card(colAmber, "Invalid usage", []string{textRow(colInk, true, "message body")})
	lines := strings.Split(got, "\n")
	if len(lines) < 4 {
		t.Fatalf("card too short:\n%s", got)
	}
	if !strings.Contains(lines[0], "INVALID USAGE") {
		t.Errorf("title tab missing/not uppercased: %q", lines[0])
	}
	if !strings.Contains(lines[0], "╭─") || !strings.Contains(lines[0], "╮") {
		t.Errorf("top border malformed: %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "╰") || !strings.Contains(lines[len(lines)-1], "╯") {
		t.Errorf("bottom border malformed: %q", lines[len(lines)-1])
	}
	if !strings.Contains(got, "message body") {
		t.Errorf("body missing:\n%s", got)
	}
}

func TestLinkify(t *testing.T) {
	got := linkify("see https://example.com/docs for more")
	if !strings.Contains(got, "example.com/docs") {
		t.Errorf("visible URL lost: %q", got)
	}
	if !strings.Contains(got, "\x1b]8;;") {
		t.Errorf("OSC-8 hyperlink escapes missing: %q", got)
	}
	if plain := linkify("no urls here"); plain != "no urls here" {
		t.Errorf("text without URLs must pass through: %q", plain)
	}
}
