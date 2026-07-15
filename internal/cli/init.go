package cli

import (
	"github.com/spf13/cobra"
)

// newInitCmd is the historical `gayle init`: by v5 it only loaded and
// validated the settings (per-provider setup was long gone). Pipelines still
// call it, so it stays — as a settings check.
func newInitCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize gayle. Only required to run once.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := d.Settings(cmd.Context(), nil); err != nil {
				return userErr(err)
			}
			logDone()
			return nil
		},
	}
}
