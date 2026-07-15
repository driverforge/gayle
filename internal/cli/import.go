package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/ui"
)

func newImportCmd(d *deps) *cobra.Command {
	var flagPath string
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import all of the configuration from the json from to a provider",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			filePath := flagPath
			if filePath == "" {
				filePath = defaultExportPath
			}

			s, err := d.Settings(ctx, nil)
			if err != nil {
				return userErr(err)
			}
			store, err := d.Store(ctx, s)
			if err != nil {
				return userErr(err)
			}

			ui.Log(ui.White("Getting parameters from: " + filePath))

			contents, err := os.ReadFile(filePath)
			if err != nil {
				return clierr.WrapT(err, "Import failed", fmt.Sprintf("could not read %s: %v", filePath, err), "")
			}
			var payload struct {
				Configs map[string]string `json:"configs"`
				Secrets map[string]string `json:"secrets"`
			}
			if err := json.Unmarshal(contents, &payload); err != nil {
				return clierr.WrapT(err, "Import failed", fmt.Sprintf("could not parse %s: %v", filePath, err), "")
			}

			configPath, secretPath := "", ""
			if s.Config != nil {
				configPath = s.Config.Path
			}
			if s.Secret != nil {
				secretPath = s.Secret.Path
			}
			// An empty section is a no-op (the Node CLI crashed on it).
			if err := importSection(ctx, store, configPath, payload.Configs, "config", updateConfigs); err != nil {
				return err
			}
			if err := importSection(ctx, store, secretPath, payload.Secrets, "secret", updateSecrets); err != nil {
				return err
			}

			ui.Log(ui.White("Saved parameters to provider"))
			logDone()
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagPath, "path", "p", "", `The location of the secrets and configuration file (default: "/tmp/gayle-exports.json")`)
	return cmd
}

func importSection(ctx context.Context, store paramstore.Store, path string, values map[string]string, section string, update func(context.Context, paramstore.Store, map[string]string) error) error {
	if len(values) == 0 {
		return nil
	}
	// The Node CLI would happily write to "undefined/KEY" here.
	if path == "" {
		return clierr.User(fmt.Sprintf("%s.path must be set in gayle.yml to import %ss", section, section), "")
	}
	parameters := make(map[string]string, len(values))
	for k, v := range values {
		parameters[path+"/"+k] = v
	}
	return update(ctx, store, parameters)
}
