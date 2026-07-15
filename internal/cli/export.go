package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/ui"
)

const defaultExportPath = "/tmp/gayle-exports.json"

func newExportCmd(d *deps) *cobra.Command {
	var (
		flagPath       string
		flagTarget     string
		flagConfigOnly bool
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export of all of the configuration from the provider to a text json file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			target := flagTarget
			if target == "" {
				target = "json"
			}
			if target != "json" && target != "env" {
				return usageError(cmd, fmt.Errorf("invalid --target %q: available options are json|env", flagTarget))
			}
			filePath := flagPath
			if filePath == "" {
				if target == "env" {
					filePath = ".env_gayle"
				} else {
					filePath = defaultExportPath
				}
			}

			s, err := d.Settings(ctx, nil)
			if err != nil {
				return userErr(err)
			}
			store, err := d.Store(ctx, s)
			if err != nil {
				return userErr(err)
			}

			ui.Log(ui.White("Getting parameters.."))

			configValues, err := store.GetParameters(ctx, s.ConfigParameters)
			if err != nil {
				return userErr(err)
			}
			configs := paramstore.ShortKeys(configValues)

			secrets := map[string]string{}
			if !flagConfigOnly {
				secretValues, err := store.GetParameters(ctx, s.SecretParameters)
				if err != nil {
					return userErr(err)
				}
				secrets = paramstore.ShortKeys(secretValues)
			}

			ui.Log(ui.White("Saving parameters to: " + filePath))

			var contents []byte
			if target == "env" {
				contents = []byte(envExport(configs, secrets))
			} else {
				contents, err = jsonExport(configs, secrets)
				if err != nil {
					return err
				}
			}
			if err := os.WriteFile(filePath, contents, 0o644); err != nil {
				return clierr.WrapT(err, "Export failed", fmt.Sprintf("could not write %s: %v", filePath, err), "")
			}
			logDone()
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagPath, "path", "p", "", `The location for the output secrets & configuration file (default: "/tmp/gayle-exports.json" or ".env_gayle")`)
	cmd.Flags().StringVarP(&flagTarget, "target", "t", "", "The output target, available options are json|env (default:json)")
	cmd.Flags().BoolVarP(&flagConfigOnly, "config-only", "C", false, "Only export configs")
	return cmd
}

// jsonExport is the Node shape byte-for-byte: {"configs": {...}, "secrets":
// {...}}, two-space indent, keys sorted, no trailing newline.
func jsonExport(configs, secrets map[string]string) ([]byte, error) {
	payload := struct {
		Configs map[string]string `json:"configs"`
		Secrets map[string]string `json:"secrets"`
	}{Configs: configs, Secrets: secrets}
	contents, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}
	return contents, nil
}

var envKeySanitizer = regexp.MustCompile(`[^A-Za-z0-9]+`)

// envExport writes KEY="value" lines (secrets win over configs on short-key
// collisions — Node merge order), keys sanitized to [A-Za-z0-9_]. Values are
// deliberately NOT escaped, matching the Node output that existing consumers
// parse; a value containing '"' or a newline breaks the file, same as v5.
func envExport(configs, secrets map[string]string) string {
	merged := map[string]string{}
	for k, v := range configs {
		merged[k] = v
	}
	for k, v := range secrets {
		merged[k] = v
	}
	var b strings.Builder
	for _, key := range sortedKeys(merged) {
		b.WriteString(envKeySanitizer.ReplaceAllString(key, "_"))
		b.WriteString(`="`)
		b.WriteString(merged[key]) // raw, unescaped — Node parity
		b.WriteString("\"\n")
	}
	return b.String()
}
