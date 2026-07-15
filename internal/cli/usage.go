package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
)

// usageError wraps a cobra command-usage error (unknown command/flag, wrong
// argument count) as an expected clierr.UserError with a pointer at the help
// text — so misuse renders as a friendly notice and exits 1, never 0.
func usageError(cmd *cobra.Command, err error) error {
	return clierr.WrapT(err, "Invalid usage", capitalise(err.Error()), "run `"+helpInvocation(cmd)+"`")
}

// helpInvocation is the help command for cmd: the per-command help for a real
// subcommand (e.g. "gayle run --help"), or the root help.
func helpInvocation(cmd *cobra.Command) string {
	if name := topLevelName(cmd); name != "" {
		return "gayle " + name + " --help"
	}
	return "gayle --help"
}

// topLevelName returns the first command segment beneath root (e.g. "run"),
// or "" for the root itself.
func topLevelName(cmd *cobra.Command) string {
	if cmd == nil || !cmd.HasParent() {
		return ""
	}
	c := cmd
	for c.Parent().HasParent() {
		c = c.Parent()
	}
	return c.Name()
}

// installUsageErrors makes cobra usage errors friendly across the whole command
// tree: flag errors via FlagErrorFunc (inherited from root), and argument-count
// / "unknown command" errors by wrapping each command's Args validator.
func installUsageErrors(root *cobra.Command) {
	root.SetFlagErrorFunc(usageError)
	walk(root, func(c *cobra.Command) {
		if c.Args == nil {
			return // a non-runnable parent; cobra raises its own error (see friendlyUsage)
		}
		validate := c.Args
		c.Args = func(cmd *cobra.Command, args []string) error {
			if err := validate(cmd, args); err != nil {
				return usageError(cmd, err)
			}
			return nil
		}
	})
}

// friendlyUsage catches the one usage error that no hook covers: cobra raises a
// bare "unknown command …" for an unrecognised subcommand before any Args
// validator runs. cmd is the resolved parent (root for `gayle foo`).
func friendlyUsage(cmd *cobra.Command, err error) error {
	if err == nil || clierr.IsUser(err) {
		return err
	}
	if strings.HasPrefix(err.Error(), "unknown command") {
		return usageError(cmd, err)
	}
	return err
}

func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, child := range c.Commands() {
		walk(child, fn)
	}
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:] // cobra messages are ASCII
}
