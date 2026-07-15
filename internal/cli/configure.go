package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/settings"
	"github.com/driverforge/gayle/internal/ui"
)

// configure is the heart of `gayle run`: populate configs, then secrets.
func configure(ctx context.Context, d *deps, s *settings.Settings, interactive, missingOnly bool) error {
	mode := "non-interactive"
	if interactive {
		mode = "interactive"
	}
	ui.Log(ui.Cyan(fmt.Sprintf("Running Gayle in %s mode..", mode)))

	store, err := d.Store(ctx, s)
	if err != nil {
		return userErr(err)
	}
	if err := populateConfig(ctx, store, s, interactive, missingOnly); err != nil {
		return err
	}
	return populateSecret(ctx, store, s, interactive, missingOnly)
}

func populateConfig(ctx context.Context, store paramstore.Store, s *settings.Settings, interactive, missingOnly bool) error {
	if s.Config == nil {
		ui.Log(ui.White("Skipping population of config..."))
		return nil
	}
	ui.Log(ui.White("Populating configuration..."))

	if s.Config.StageOverrides == nil {
		ui.Log(ui.Cyan("Could not find configuration overrides. Using default values."))
	}
	if s.Config.Path == "" {
		return clierr.User("'config.path' must be set in gayle.yml to populate configs.", "")
	}

	prompted, err := promptRequired(ctx, store, s.ConfigParameters, s.Config.Required, interactive, missingOnly)
	if err != nil {
		return err
	}

	// Merge precedence (later wins): prompted < defaults < stage overrides.
	// Defaults overriding freshly prompted values is surprising but
	// load-bearing Node behavior — pipelines rely on defaults winning.
	// (The Node `outputs` layer between defaults and overrides was dead code:
	// nothing ever set settings.outputs.)
	merged := map[string]string{}
	for k, v := range prompted {
		merged[k] = v
	}
	for k, v := range s.Config.Defaults {
		merged[k] = v
	}
	for k, v := range s.Config.StageOverrides {
		merged[k] = v
	}

	parameters := make(map[string]string, len(merged))
	for k, v := range merged {
		parameters[s.Config.Path+"/"+k] = v
	}
	return updateConfigs(ctx, store, parameters)
}

func populateSecret(ctx context.Context, store paramstore.Store, s *settings.Settings, interactive, missingOnly bool) error {
	if s.Secret == nil {
		ui.Log(ui.White("Skipping population of secrets..."))
		return nil
	}
	ui.Log(ui.White("Populating secrets..."))

	if s.Secret.Path == "" {
		return clierr.User("'secret.path' must be set in gayle.yml to populate secrets.", "")
	}

	secrets, err := promptRequired(ctx, store, s.SecretParameters, s.Secret.Required, interactive, missingOnly)
	if err != nil {
		return err
	}

	parameters := make(map[string]string, len(secrets))
	for k, v := range secrets {
		parameters[s.Secret.Path+"/"+k] = v
	}
	return updateSecrets(ctx, store, parameters)
}

// promptRequired resolves the required keys' values: non-interactively it
// verifies they already exist remotely (warning per missing key, then a hard
// error); interactively it prompts — all keys, or with missingOnly just the
// unset ones.
func promptRequired(ctx context.Context, store paramstore.Store, parameterNames []string, required map[string]string, interactive, missingOnly bool) (map[string]string, error) {
	current, err := store.GetParameters(ctx, parameterNames)
	if err != nil {
		return nil, userErr(err)
	}
	currentByKey := paramstore.ShortKeys(current)

	if !interactive {
		return validateExisting(required, currentByKey)
	}

	if missingOnly {
		missing := map[string]string{}
		for key, desc := range required {
			if currentByKey[key] == "" {
				missing[key] = desc
			}
		}
		return promptForValues(missing, nil)
	}
	return promptForValues(required, currentByKey)
}

// validateExisting is the non-interactive path: every required key must
// already hold a remote value; the run never invents values on its own.
func validateExisting(required, current map[string]string) (map[string]string, error) {
	configs := map[string]string{}
	for _, key := range sortedKeys(required) {
		if current[key] == "" {
			ui.Warn(fmt.Sprintf("Config missing: [%s: %s]", key, required[key]))
			continue
		}
		configs[key] = current[key]
	}
	if len(configs) != len(required) {
		return nil, clierr.User("Missing required configs!! Run on interactive mode to populate them!!", "")
	}
	return configs, nil
}

// promptForValues asks for each required key in turn, pre-filled with its
// current value. Keys prompt in sorted order (the yml's declaration order is
// not preserved by the parser).
func promptForValues(required, current map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range sortedKeys(required) {
		value, err := ui.PromptValue(key, required[key], current[key])
		if err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, nil
}

// updateConfigs / updateSecrets wrap the store writes in the Node CLI's
// progress logs. Secrets log only the value length.
func updateConfigs(ctx context.Context, store paramstore.Store, parameters map[string]string) error {
	ui.Log(ui.Gray("Updating config..."))
	results, err := store.PutConfigs(ctx, parameters)
	for _, r := range results {
		ui.Log(ui.Gray(fmt.Sprintf("Updated config: Name: %s | Value: [%s] | Version: [%s]", r.Name, r.Value, r.Version)))
	}
	if err != nil {
		return clierr.WrapT(err, "Update failed", err.Error(), "")
	}
	ui.Log(ui.Gray("Config updated"))
	return nil
}

func updateSecrets(ctx context.Context, store paramstore.Store, parameters map[string]string) error {
	ui.Log(ui.Gray("Updating secrets..."))
	results, err := store.PutSecrets(ctx, parameters)
	for _, r := range results {
		ui.Log(ui.Gray(fmt.Sprintf("Updated secret: Name: %s | Value: [%d chars] | Version: [%s]", r.Name, len(r.Value), r.Version)))
	}
	if err != nil {
		return clierr.WrapT(err, "Update failed", err.Error(), "")
	}
	ui.Log(ui.Gray("Secrets updated"))
	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// userErr wraps an anticipated domain error (settings load, provider reads)
// as a UserError so it exits 1 with a friendly render instead of a crash card.
func userErr(err error) error {
	if err == nil || clierr.IsUser(err) {
		return err
	}
	return clierr.Wrap(err, err.Error(), "")
}

// logDone prints the Node CLI's green success footer.
func logDone() {
	ui.Log(ui.Green("Done."))
}
