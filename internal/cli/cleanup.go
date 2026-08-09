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
			if err := cleanUp(ctx, d, s, flagDryRun, populateBoth); err != nil {
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
// Also invoked by `run -r` (which never dry-runs — Node parity), passing that
// run's populateMode so a restricted run prunes only the half it populated.
func cleanUp(ctx context.Context, d *deps, s *settings.Settings, dryRun bool, mode populateMode) error {
	configDeclared := append([]string{}, s.ConfigParameters...)
	// Stage-override keys are absent from ConfigParameters (v5 parity, see
	// derive.go) but populateConfig writes them — cleanup must count them
	// as declared or `run -r` deletes parameters it wrote moments earlier
	// (DF-659).
	if s.Config != nil {
		for k := range s.Config.StageOverrides {
			configDeclared = append(configDeclared, s.Config.Path+"/"+k)
		}
	}
	secretDeclared := append([]string{}, s.SecretParameters...)
	// Everything declared is protected whatever the mode; the mode narrows
	// which paths are listed for orphans, never which keys are safe.
	declared := append(append([]string{}, configDeclared...), secretDeclared...)

	// DF-644 guard: an empty or misparsed configuration would classify every
	// remote parameter under the configured paths as unused and delete the lot.
	// The guard applies to the half being pruned — a config-only prune against
	// a config block that declares nothing is the same hazard.
	scoped, what := declared, "config or secret"
	switch mode {
	case populateConfigsOnly:
		scoped, what = configDeclared, "config"
	case populateSecretsOnly:
		scoped, what = secretDeclared, "secret"
	}
	if len(scoped) == 0 {
		return clierr.User(fmt.Sprintf("Cleanup refused: the configuration declares no %s keys. ", what)+
			"Pruning against an empty declaration would delete every remote parameter under the configured paths.", "")
	}

	configPath, secretPath := "", ""
	if s.Config != nil {
		configPath = s.Config.Path
	}
	if s.Secret != nil {
		secretPath = s.Secret.Path
	}
	switch mode {
	case populateConfigsOnly:
		if configPath == "" {
			return clierr.User("Cleanup requires 'config.path' to be set in gayle.yml.", "")
		}
	case populateSecretsOnly:
		if secretPath == "" {
			return clierr.User("Cleanup requires 'secret.path' to be set in gayle.yml.", "")
		}
	default:
		// Cleanup reads both paths (Node parity) and errors when either is unset.
		if configPath == "" || secretPath == "" {
			return clierr.User("Cleanup requires both 'config.path' and 'secret.path' to be set in gayle.yml.", "")
		}
	}
	// A shared path (the norm for Key Vault) makes "orphan config" and "orphan
	// secret" indistinguishable by name, so a restricted prune cannot keep its
	// promise to leave the other half alone. Refuse rather than guess.
	if mode != populateBoth && configPath != "" && configPath == secretPath {
		return clierr.UserT("Cleanup scope is ambiguous",
			fmt.Sprintf("%s cannot prune orphans: 'config.path' and 'secret.path' are both %s, so an orphan cannot be attributed to configs or secrets.", mode.flag(), configPath),
			"drop --removing, or give configs and secrets separate paths")
	}

	store, err := d.Store(ctx, s)
	if err != nil {
		return userErr(err)
	}

	var remote []paramstore.Parameter
	if mode.configs() {
		configs, err := store.GetAllByPath(ctx, configPath)
		if err != nil {
			return userErr(err)
		}
		remote = configs
	}
	// Key Vault declarations routinely share one path for config and
	// secrets; listing it twice queued every orphan for a double delete
	// whose second attempt 404'd (DF-659).
	if mode.secrets() && secretPath != configPath {
		secrets, err := store.GetAllByPath(ctx, secretPath)
		if err != nil {
			return userErr(err)
		}
		remote = append(remote, secrets...)
	}

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
