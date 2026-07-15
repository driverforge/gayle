package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/ui"
)

func newListCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all remote configurations and secrets.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := d.Settings(ctx, nil)
			if err != nil {
				return userErr(err)
			}
			store, err := d.Store(ctx, s)
			if err != nil {
				return userErr(err)
			}

			ui.Log(ui.White("Listing all configurations.."))

			ui.Log(ui.Cyan("Configs:"))
			if err := listParameters(ctx, store, s.ConfigParameters, false); err != nil {
				return err
			}
			ui.Log(ui.Cyan("Secrets:"))
			if err := listParameters(ctx, store, s.SecretParameters, true); err != nil {
				return err
			}
			logDone()
			return nil
		},
	}
}

func listParameters(ctx context.Context, store paramstore.Store, names []string, mask bool) error {
	values, err := store.GetParameters(ctx, names)
	if err != nil {
		return userErr(err)
	}
	byKey := paramstore.ShortKeys(values)
	for _, key := range sortedKeys(byKey) {
		value := byKey[key]
		if mask {
			value = ui.Mask(value)
		}
		ui.Log(ui.Gray(fmt.Sprintf("  %s: %s", key, value)))
	}
	return nil
}
