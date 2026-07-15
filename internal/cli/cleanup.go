package cli

import (
	"context"
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/settings"
	"github.com/driverforge/gayle/internal/ui"
)

func newCleanUpCmd(d *deps) *cobra.Command {
	var flagDryRun bool
	cmd := &cobra.Command{
		Use:   "clean-up",
		Short: "Cleaning up orphan configs or secrets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := d.Settings(ctx, nil)
			if err != nil {
				return userErr(err)
			}
			if err := cleanUp(ctx, d, s, flagDryRun); err != nil {
				return err
			}
			logDone()
			return nil
		},
	}
	cmd.Flags().BoolVarP(&flagDryRun, "dry-run", "d", false, "Execute a dry run")
	return cmd
}

// cleanUp prunes remote parameters that no longer appear in the declaration.
// Also invoked by `run -r` (which never dry-runs — Node parity).
func cleanUp(ctx context.Context, d *deps, s *settings.Settings, dryRun bool) error {
	declared := append(append([]string{}, s.ConfigParameters...), s.SecretParameters...)

	// DF-644 guard: an empty or misparsed configuration would classify every
	// remote parameter under the configured paths as unused and delete the lot.
	if len(declared) == 0 {
		return clierr.User("Cleanup refused: the configuration declares no config or secret keys. "+
			"Pruning against an empty declaration would delete every remote parameter under the configured paths.", "")
	}

	configPath, secretPath := "", ""
	if s.Config != nil {
		configPath = s.Config.Path
	}
	if s.Secret != nil {
		secretPath = s.Secret.Path
	}
	// Cleanup reads both paths (Node parity) and errors when either is unset.
	if configPath == "" || secretPath == "" {
		return clierr.User("Cleanup requires both 'config.path' and 'secret.path' to be set in gayle.yml.", "")
	}

	store, err := d.Store(ctx, s)
	if err != nil {
		return userErr(err)
	}

	configs, err := store.GetAllByPath(ctx, configPath)
	if err != nil {
		return userErr(err)
	}
	secrets, err := store.GetAllByPath(ctx, secretPath)
	if err != nil {
		return userErr(err)
	}
	remote := append(configs, secrets...)

	var unused []paramstore.Parameter
	for _, p := range remote {
		if !slices.Contains(declared, p.Name) {
			unused = append(unused, p)
		}
	}

	if len(unused) == 0 {
		ui.Log(ui.Gray("Cleanup --> No unused parameters"))
		return nil
	}

	if dryRun {
		ui.Log(ui.Gray("Cleanup --> Parameters to be deleted: "))
		for _, p := range unused {
			value := p.Value
			if p.Type == paramstore.TypeSecureString {
				value = ui.Mask(value)
			}
			ui.Log(ui.Gray(fmt.Sprintf("Cleanup --> Name: %s | Value: [%s]", p.Name, value)))
		}
		return nil
	}

	names := make([]string, len(unused))
	for i, p := range unused {
		ui.Log(ui.Yellow("Cleanup --> Deleting unused parameter: " + p.Name))
		names[i] = p.Name
	}
	ui.Log(ui.Gray("Cleanup --> Deleting unused parameters..."))
	if err := store.DeleteParameters(ctx, names); err != nil {
		return clierr.WrapT(err, "Cleanup failed", err.Error(), "")
	}
	ui.Log(ui.Gray("Cleanup --> All orphan parameters deleted"))
	return nil
}
