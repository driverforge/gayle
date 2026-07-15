package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/settings"
)

// newGenerateCmd writes the example gayle.yml. It is the one command that
// needs no --stage. The write is checked — the Node CLI fired an async write
// with a swallowed error and could print Done. over a missing file.
func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate an example configuration file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, _ := os.Getwd()
			if _, err := os.Stat("gayle.yml"); err == nil {
				return clierr.User(fmt.Sprintf("gayle.yml file already exists in the following directory -- %s", cwd), "")
			}
			if err := os.WriteFile("gayle.yml", []byte(settings.GenerateTemplate), 0o644); err != nil {
				return clierr.WrapT(err, "Generate failed", fmt.Sprintf("could not write gayle.yml: %v", err), "")
			}
			logDone()
			return nil
		},
	}
}
