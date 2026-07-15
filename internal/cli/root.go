// Package cli wires the Cobra command tree for gayle. Commands are thin
// adapters: they parse flags and call the domain packages (internal/settings,
// internal/paramstore). The CLI surface — command names, flags, defaults, log
// wording, and the stdout/stderr split — deliberately matches the Node CLI it
// replaced, because CI pipelines invoke it unchanged.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/buildinfo"
	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/ui"
)

var (
	flagStage  string
	flagConfig string
)

// Execute runs the root command and returns a process exit code: 0 on verified
// success, 1 for expected failures (clierr.UserError — bad usage, missing
// values, provider write failures), 2 for a crash. The exit code is the
// contract CI pipelines gate on, so an error can never map to 0.
func Execute() (code int) {
	defer func() {
		if r := recover(); r != nil {
			ui.RenderCrash(os.Stderr, fmt.Errorf("gayle crashed: %v", r))
			code = 2
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := newRootCmd()
	cmd, err := root.ExecuteContextC(ctx)
	if err != nil {
		// Cobra usage errors (unknown command/flag, bad args) arrive here
		// unwrapped; make them friendly. cmd is the command cobra resolved.
		err = friendlyUsage(cmd, err)
		if !clierr.IsUser(err) {
			ui.RenderCrash(os.Stderr, err)
			return 2
		}
		ui.RenderUserError(os.Stderr, err)
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gayle",
		Short:         "Deploy configuration and secrets to AWS SSM Parameter Store or Azure Key Vault",
		SilenceUsage:  true,
		SilenceErrors: true,
		// Version enables the --version flag; the -V shorthand is registered
		// below to match the Node CLI (commander's default).
		Version: buildinfo.Version,
		// A bare `gayle` prints help but still fails — a pipeline that lost its
		// arguments must not exit 0. (The Node CLI intended this too, but
		// commander's help() exited 0 first.)
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return clierr.Silent()
		},
		// Every command except generate (and help/completion/version) operates
		// on a stage. Validated here so the message is uniform.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if requiresStage(cmd) && flagStage == "" {
				return clierr.UserT("Missing stage",
					"Invalid options!! You must specify stage.",
					"pass -s <stage>")
			}
			return nil
		},
	}

	root.PersistentFlags().StringVarP(&flagStage, "stage", "s", "", "Specify stage to run on. (required)")
	root.PersistentFlags().StringVarP(&flagConfig, "config", "c", "gayle.yml", "Path to gayle configuration")
	// Register the version flag ourselves so it gets the -V shorthand (cobra's
	// default -v would collide with run's --variables shorthand mnemonic).
	root.Flags().BoolP("version", "V", false, "output the version number")
	root.SetVersionTemplate(buildinfo.String() + "\n")

	root.AddCommand(
		newRunCmd(),
		newInitCmd(),
		newGenerateCmd(),
		newExportCmd(),
		newImportCmd(),
		newListCmd(),
		newFetchCmd(),
		newCleanUpCmd(),
	)
	installUsageErrors(root) // make unknown command/flag/arg errors friendly
	return root
}

// requiresStage reports whether cmd needs --stage: everything except generate
// and the meta commands (help, completion, and the root itself, which only
// prints help).
func requiresStage(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "generate", "help", "completion", "gayle":
		return false
	}
	if p := cmd.Parent(); p != nil && p.Name() == "completion" {
		return false
	}
	return true
}
