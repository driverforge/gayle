package cli

// The CLI-surface pin test: gayle's Go port must expose exactly the command and
// flag surface of the Node CLI (v5.6.0), because CI pipelines invoke it
// unchanged. A failure here means a pipeline-visible break — change with care
// and a CHANGELOG entry.

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
)

func resetFlags() {
	flagStage = ""
	flagConfig = "gayle.yml"
}

func execute(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetFlags()
	root := newRootCmd(newDeps())
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(args)
	cmd, err := root.ExecuteC()
	// Mirror Execute(): unknown-command errors are made friendly at the top.
	err = friendlyUsage(cmd, err)
	return out.String(), errb.String(), err
}

func TestCommandSurface(t *testing.T) {
	root := newRootCmd(newDeps())
	want := []string{"run", "init", "generate", "export", "import", "list", "fetch", "clean-up"}
	byName := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		byName[c.Name()] = c
	}
	for _, name := range want {
		if byName[name] == nil {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestFlagSurface(t *testing.T) {
	root := newRootCmd(newDeps())

	// name → flag name → {shorthand, default}
	type f struct{ short, def string }
	wantPersistent := map[string]f{
		"stage":  {"s", ""},
		"config": {"c", "gayle.yml"},
	}
	for name, spec := range wantPersistent {
		fl := root.PersistentFlags().Lookup(name)
		if fl == nil {
			t.Fatalf("missing persistent flag --%s", name)
		}
		if fl.Shorthand != spec.short || fl.DefValue != spec.def {
			t.Errorf("--%s: shorthand=%q default=%q, want %q/%q", name, fl.Shorthand, fl.DefValue, spec.short, spec.def)
		}
	}

	if fl := root.Flags().Lookup("version"); fl == nil || fl.Shorthand != "V" {
		t.Errorf("root --version flag must exist with shorthand -V (commander compat)")
	}

	wantCmd := map[string]map[string]f{
		"run": {
			"variables":   {"v", ""},
			"interactive": {"i", "false"},
			"missing":     {"m", "false"},
			"removing":    {"r", "false"},
		},
		"export": {
			"path":        {"p", ""},
			"target":      {"t", ""},
			"config-only": {"C", "false"},
		},
		"import":   {"path": {"p", ""}},
		"fetch":    {"keys": {"k", ""}},
		"clean-up": {"dry-run": {"d", "false"}},
	}
	for _, c := range root.Commands() {
		spec, ok := wantCmd[c.Name()]
		if !ok {
			continue
		}
		for name, want := range spec {
			fl := c.Flags().Lookup(name)
			if fl == nil {
				t.Errorf("%s: missing flag --%s", c.Name(), name)
				continue
			}
			if fl.Shorthand != want.short || fl.DefValue != want.def {
				t.Errorf("%s --%s: shorthand=%q default=%q, want %q/%q", c.Name(), name, fl.Shorthand, fl.DefValue, want.short, want.def)
			}
		}
	}
}

// Usage misuse must produce an error (→ exit 1); the Node CLI accidentally
// exited 0 via commander's help().
func TestUsageErrorsAreErrors(t *testing.T) {
	cases := [][]string{
		{},                    // bare gayle: help + fail
		{"bogus"},             // unknown command
		{"list"},              // missing --stage
		{"run", "--nonsense"}, // unknown flag
		{"list", "-s"},        // -s without a value
	}
	for _, args := range cases {
		_, _, err := execute(t, args...)
		if err == nil {
			t.Errorf("args %v: expected an error, got nil", args)
			continue
		}
		if !clierr.IsUser(err) {
			t.Errorf("args %v: error should be a UserError (exit 1, not a crash): %v", args, err)
		}
	}
}

func TestMissingStageMessage(t *testing.T) {
	_, _, err := execute(t, "list")
	if err == nil || err.Error() != "Invalid options!! You must specify stage. (pass -s <stage>)" {
		t.Errorf("missing-stage error mismatch: %v", err)
	}
}

// generate is the one command that must work without --stage.
func TestGenerateNeedsNoStage(t *testing.T) {
	t.Chdir(t.TempDir()) // generate writes gayle.yml into the working directory
	_, _, err := execute(t, "generate")
	if err != nil && err.Error() == "Invalid options!! You must specify stage. (pass -s <stage>)" {
		t.Errorf("generate must not require --stage")
	}
}

// Global flags are accepted after the subcommand (commander compat:
// `gayle run -s dev` and `gayle -s dev run` are both valid).
func TestGlobalFlagsAfterSubcommand(t *testing.T) {
	_, _, err := execute(t, "list", "-s", "dev")
	if err != nil && err.Error() == "Invalid options!! You must specify stage. (pass -s <stage>)" {
		t.Errorf("-s after subcommand not honoured: %v", err)
	}
}

func TestVersionFlag(t *testing.T) {
	out, _, err := execute(t, "-V")
	if err != nil {
		t.Fatalf("-V: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("gayle ")) {
		t.Errorf("-V output missing version line: %q", out)
	}
}
