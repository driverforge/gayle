package cli

// Stub implementations of the eight gayle commands. Each declares its final
// flag surface (mirroring the Node CLI) so the surface pin test holds from the
// first commit; the bodies are ported in later commits.

import (
	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
)

func notImplemented(cmd *cobra.Command, _ []string) error {
	return clierr.UserT("Not implemented", "`gayle "+cmd.Name()+"` has not been ported yet.", "")
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Verify or populate all remote configurations and secrets.",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
	cmd.Flags().StringP("variables", "v", "", "Variables used for config interpolation.")
	cmd.Flags().BoolP("interactive", "i", false, "Run on interactive mode")
	cmd.Flags().BoolP("missing", "m", false, "Only prompt missing values in interactive mode")
	cmd.Flags().BoolP("removing", "r", false, "Removing orphan configs or secrets")
	return cmd
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize gayle. Only required to run once.",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
}

func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate an example configuration file.",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
}

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export of all of the configuration from the provider to a text json file",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
	cmd.Flags().StringP("path", "p", "", `The location for the output secrets & configuration file (default: "/tmp/gayle-exports.json" or ".env_gayle")`)
	cmd.Flags().StringP("target", "t", "", "The output target, available options are json|env (default:json)")
	cmd.Flags().BoolP("config-only", "C", false, "Only export configs")
	return cmd
}

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import all of the configuration from the json from to a provider",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
	cmd.Flags().StringP("path", "p", "", `The location of the secrets and configuration file (default: "/tmp/gayle-exports.json")`)
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all remote configurations and secrets.",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
}

func newFetchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch config or secret",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
	cmd.Flags().StringP("keys", "k", "", `Comma separated configs to fetch (example: "SOME_CONFIG,ANOTHER_CONFIG")`)
	return cmd
}

func newCleanUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-up",
		Short: "Cleaning up orphan configs or secrets",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
	cmd.Flags().BoolP("dry-run", "d", false, "Execute a dry run")
	return cmd
}
