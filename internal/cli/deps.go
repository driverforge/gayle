package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/paramstore/keyvault"
	"github.com/driverforge/gayle/internal/paramstore/ssm"
	"github.com/driverforge/gayle/internal/settings"
	"github.com/driverforge/gayle/internal/ui"
)

// deps carries the two seams every command works through — settings loading
// and provider store construction — memoized per run. It replaces the Node
// CLI's memoized settings service and DataLoader wiring; tests swap load /
// newStore for fixtures and the in-memory fake.
type deps struct {
	load     func(ctx context.Context, configPath string, vars map[string]string, stage string) (*settings.Settings, error)
	newStore func(ctx context.Context, s *settings.Settings) (paramstore.Store, error)

	mu       sync.Mutex
	settings *settings.Settings
	store    paramstore.Store
}

func newDeps() *deps {
	return &deps{
		load: func(ctx context.Context, configPath string, vars map[string]string, stage string) (*settings.Settings, error) {
			return settings.Loader{}.Load(ctx, configPath, vars, stage)
		},
		newStore: defaultNewStore,
	}
}

// Settings loads (once) the gayle.yml for this run, printing the Node CLI's
// stage/config banner first. vars are the run command's -v variables; other
// commands pass nil.
func (d *deps) Settings(ctx context.Context, vars map[string]string) (*settings.Settings, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.settings != nil {
		return d.settings, nil
	}
	configPath, err := filepath.Abs(flagConfig)
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}
	ui.Log(ui.Gray("Stage  --> " + flagStage))
	ui.Log(ui.Gray("Config --> " + configPath))
	s, err := d.load(ctx, configPath, vars, flagStage)
	if err != nil {
		return nil, err
	}
	if s.Secret != nil && s.Secret.KeyID != "" {
		// v5 documented secret.keyId but never honored it; SSM secrets are
		// always encrypted with alias/aws/ssm. Honoring it now would silently
		// re-encrypt existing parameters, so it stays ignored — loudly.
		ui.Warn("secret.keyId is set but not supported: SSM secrets are always encrypted with alias/aws/ssm")
	}
	d.settings = s
	return s, nil
}

// Store returns (once) the provider store the settings select.
func (d *deps) Store(ctx context.Context, s *settings.Settings) (paramstore.Store, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.store != nil {
		return d.store, nil
	}
	st, err := d.newStore(ctx, s)
	if err != nil {
		return nil, err
	}
	d.store = st
	return st, nil
}

func defaultNewStore(ctx context.Context, s *settings.Settings) (paramstore.Store, error) {
	switch s.Provider.Name {
	case "ssm":
		return ssm.New(ctx)
	case "key-vault":
		return keyvault.New(s.Provider.Vault)
	default:
		// Unreachable after settings validation; kept as the Node message.
		return nil, fmt.Errorf("Unsupported provider specified")
	}
}
