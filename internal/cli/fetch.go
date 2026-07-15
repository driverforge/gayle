package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/paramstore"
)

// newFetchCmd prints the requested keys as JSON on STDOUT — the only stdout
// output in the whole tool; every log line stays on stderr so
// `gayle fetch ... | jq` keeps working.
func newFetchCmd(d *deps) *cobra.Command {
	var flagKeys string
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch config or secret",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if flagKeys == "" {
				// The Node CLI crashed on a missing -k; make it a usage error.
				return usageError(cmd, fmt.Errorf("required flag --keys not set"))
			}

			s, err := d.Settings(ctx, nil)
			if err != nil {
				return userErr(err)
			}
			store, err := d.Store(ctx, s)
			if err != nil {
				return userErr(err)
			}

			declared := append(append([]string{}, s.ConfigParameters...), s.SecretParameters...)

			var names []string
			var unknown []string
			for _, key := range strings.Split(flagKeys, ",") {
				name := ""
				for _, param := range declared {
					if paramstore.ShortKey(param) == key {
						name = param
						break
					}
				}
				if name == "" {
					// The Node CLI silently dropped undeclared keys, shrinking
					// the JSON without a word — an error is honest.
					unknown = append(unknown, key)
					continue
				}
				names = append(names, name)
			}
			if len(unknown) > 0 {
				return clierr.UserT("Unknown keys",
					fmt.Sprintf("Not declared in the configuration: %s", strings.Join(unknown, ", ")),
					"keys must appear under config or secret in gayle.yml")
			}

			values, err := store.GetParameters(ctx, names)
			if err != nil {
				return userErr(err)
			}
			output, err := json.MarshalIndent(paramstore.ShortKeys(values), "", "  ")
			if err != nil {
				return fmt.Errorf("fetch: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(output))
			logDone()
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagKeys, "keys", "k", "", `Comma separated configs to fetch (example: "SOME_CONFIG,ANOTHER_CONFIG")`)
	return cmd
}
