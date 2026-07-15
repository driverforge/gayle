package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/ui"
)

func newRunCmd(d *deps) *cobra.Command {
	var (
		flagVariables   string
		flagInteractive bool
		flagMissing     bool
		flagRemoving    bool
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Verify or populate all remote configurations and secrets.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			if flagInteractive && !ui.Interactive() {
				return clierr.UserT("No TTY",
					"Interactive mode requires a terminal.",
					"run without -i, or populate values from a terminal")
			}

			vars, err := parseVariables(flagVariables)
			if err != nil {
				return err
			}

			s, err := d.Settings(ctx, vars)
			if err != nil {
				return userErr(err)
			}
			// The Node `init` step ran here; it was a no-op beyond forcing the
			// settings load above.
			if err := configure(ctx, d, s, flagInteractive, flagMissing); err != nil {
				return err
			}
			if flagRemoving {
				if err := cleanUp(ctx, d, s, false); err != nil {
					return err
				}
			}
			logDone()
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagVariables, "variables", "v", "", "Variables used for config interpolation.")
	cmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "Run on interactive mode")
	cmd.Flags().BoolVarP(&flagMissing, "missing", "m", false, "Only prompt missing values in interactive mode")
	cmd.Flags().BoolVarP(&flagRemoving, "removing", "r", false, "Removing orphan configs or secrets")
	return cmd
}

// parseVariables decodes the -v JSON object into interpolation variables,
// coercing scalars to strings the way lodash template would have.
func parseVariables(raw string) (map[string]string, error) {
	if raw == "" {
		return nil, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, clierr.UserT("Invalid variables",
			fmt.Sprintf("Variables must be in JSON format!! %s.", err.Error()),
			`example: -v '{"foo":"bar"}'`)
	}
	vars := make(map[string]string, len(parsed))
	for k, v := range parsed {
		switch t := v.(type) {
		case string:
			vars[k] = t
		case float64:
			vars[k] = strconv.FormatFloat(t, 'f', -1, 64)
		case bool:
			vars[k] = strconv.FormatBool(t)
		case nil:
			vars[k] = ""
		default:
			return nil, clierr.UserT("Invalid variables",
				fmt.Sprintf("Variable %q must be a scalar value.", k), "")
		}
	}
	return vars, nil
}
