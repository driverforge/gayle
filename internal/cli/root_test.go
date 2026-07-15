package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/driverforge/gayle/internal/clierr"
)

// The exit-code contract: 1 for expected failures, 2 for bugs. CI pipelines
// gate on this — 0 must strictly mean success.
func TestRenderFailureExitCodes(t *testing.T) {
	root := newRootCmd(newDeps())

	var buf bytes.Buffer
	if code := renderFailure(root, clierr.User("expected condition", ""), &buf); code != 1 {
		t.Errorf("UserError → %d, want 1", code)
	}
	if !strings.Contains(buf.String(), "Gayle: ERROR: expected condition") {
		t.Errorf("UserError not rendered: %q", buf.String())
	}

	buf.Reset()
	if code := renderFailure(root, errors.New("nil map write"), &buf); code != 2 {
		t.Errorf("unexpected error → %d, want 2", code)
	}
	if !strings.Contains(buf.String(), "Gayle: ERROR: nil map write") {
		t.Errorf("crash not rendered: %q", buf.String())
	}

	// A wrapped UserError deep in a chain still classifies as expected.
	buf.Reset()
	wrapped := clierr.Wrap(errors.New("root cause"), "friendly face", "")
	if code := renderFailure(root, wrapped, &buf); code != 1 {
		t.Errorf("wrapped UserError → %d, want 1", code)
	}

	// Cobra's bare "unknown command" arrives unwrapped and must become a
	// friendly usage error (exit 1), not a crash.
	buf.Reset()
	if code := renderFailure(root, errors.New(`unknown command "bogus" for "gayle"`), &buf); code != 1 {
		t.Errorf("unknown command → %d, want 1", code)
	}
	if !strings.Contains(buf.String(), "Gayle: ERROR: Unknown command") {
		t.Errorf("usage error not rendered friendly: %q", buf.String())
	}
}

func TestParseVariables(t *testing.T) {
	got, err := parseVariables(`{"s":"str","n":3200,"f":3.5,"b":true,"z":null}`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"s": "str", "n": "3200", "f": "3.5", "b": "true", "z": ""}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("vars[%q] = %q, want %q", k, got[k], v)
		}
	}

	if vars, err := parseVariables(""); err != nil || vars != nil {
		t.Errorf("empty input must be a nil map, got %v, %v", vars, err)
	}

	if _, err := parseVariables("{oops"); err == nil || !strings.Contains(err.Error(), "Variables must be in JSON format!!") {
		t.Errorf("invalid JSON error mismatch: %v", err)
	}

	if _, err := parseVariables(`{"nested":{"a":1}}`); err == nil || !strings.Contains(err.Error(), "scalar") {
		t.Errorf("non-scalar value must error: %v", err)
	}
}
