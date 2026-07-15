// Package settings loads, validates, interpolates, and derives the gayle.yml
// configuration — the port of the Node CLI's settings service. The pipeline
// (validate the raw tree, gather AWS context, interpolate every scalar, then
// derive the declared parameter names) and its user-visible error strings
// match the Node behavior; deviations are called out inline.
package settings

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Provider selects the parameter store backend.
type Provider struct {
	Name  string // "ssm" or "key-vault"
	Vault string // Azure Key Vault name (key-vault only)
}

// ConfigBlock is the `config:` section: plain (non-secret) parameters.
type ConfigBlock struct {
	Path     string
	Defaults map[string]string // always-applied values
	Required map[string]string // KEY → prompt description
	// StageOverrides is the block keyed by the literal stage name (e.g.
	// `production:`), already resolved for the stage this run targets. nil when
	// the stage has no override block (a different message prints in that case).
	StageOverrides map[string]string
}

// SecretBlock is the `secret:` section: SecureString/Key Vault values.
type SecretBlock struct {
	KeyID    string // documented in v5 but never honored; a warning prints if set
	Path     string
	Required map[string]string // KEY → prompt description
}

// Settings is the fully interpolated configuration plus the derived
// parameter-name lists every command works from.
type Settings struct {
	Service  string
	Provider Provider
	Stacks   []string
	Config   *ConfigBlock
	Secret   *SecretBlock

	// ConfigParameters is every declared config key (defaults ∪ required, NOT
	// stage-override-only keys — Node parity) as a full name "<path>/<KEY>",
	// sorted. SecretParameters likewise for secret.required.
	ConfigParameters []string
	SecretParameters []string
}

// Loader loads Settings. The zero value uses the real AWS SDK for the
// account/region/CloudFormation context; tests inject AWSContext.
type Loader struct {
	// AWSContext returns the interpolation variables contributed by AWS:
	// accountId, region, and every CloudFormation output of stackNames.
	// nil means the real implementation (aws.go).
	AWSContext func(ctx context.Context, stackNames []string) (map[string]string, error)
}

// Load reads and processes the gayle.yml at filePath. cliVars are the
// -v/--variables values; stage is always available as ${stage}.
func (l Loader) Load(ctx context.Context, filePath string, cliVars map[string]string, stage string) (*Settings, error) {
	raw, err := readConfig(filePath)
	if err != nil {
		return nil, err
	}
	if err := validateProvider(raw); err != nil {
		return nil, err
	}

	vars := make(map[string]string, len(cliVars)+1)
	for k, v := range cliVars {
		vars[k] = v
	}
	vars["stage"] = stage

	// Key Vault needs no AWS context at all (no AWS calls are made); for SSM the
	// account, region, and stack outputs become interpolation variables, with
	// the AWS context winning on key collisions (Node merge order).
	if providerName(raw) != "key-vault" {
		stacks, err := interpolatedStacks(raw, vars)
		if err != nil {
			return nil, err
		}
		awsCtx := l.AWSContext
		if awsCtx == nil {
			awsCtx = awsContext
		}
		extra, err := awsCtx(ctx, stacks)
		if err != nil {
			return nil, err
		}
		for k, v := range extra {
			vars[k] = v
		}
	}

	tree, err := deepMap(raw, vars)
	if err != nil {
		return nil, err
	}
	return extract(tree, stage)
}

// readConfig reads and parses the yml. A missing file keeps the Node CLI's
// message; a malformed file reports the real parse error (the Node CLI
// misreported any parse failure as "could not find" — an honesty fix).
func readConfig(filePath string) (map[string]any, error) {
	if !strings.HasSuffix(filePath, ".yml") {
		return nil, fmt.Errorf("unsupported file type %q: only .yml configuration is supported", filePath)
	}
	contents, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			return nil, fmt.Errorf("Could not find gayle.yml in the following directory - %s", cwd)
		}
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	return raw, nil
}

// validateProvider checks the RAW (pre-interpolation) tree, like the Node CLI:
// the provider name must be literal, but the vault may contain ${stage}.
func validateProvider(raw map[string]any) error {
	name := providerName(raw)
	if name != "ssm" && name != "key-vault" {
		display := name
		if display == "" {
			display = "undefined" // what lodash get + template printed for a missing name
		}
		return fmt.Errorf("Invalid provider '%s'!! Only ssm,key-vault are supported.", display)
	}
	if name == "key-vault" {
		if vault, _ := mapAt(raw, "provider")["vault"].(string); vault == "" {
			return fmt.Errorf("Invalid provider!! 'provider.vault' must be passed for 'key-vault' provider.")
		}
	}
	return nil
}

func providerName(raw map[string]any) string {
	name, _ := mapAt(raw, "provider")["name"].(string)
	return name
}

// interpolatedStacks resolves the raw `stacks:` list with the base variables
// (stage + cli vars) — stack names are interpolated BEFORE the AWS context
// exists, so they may reference ${stage} but not ${accountId} (Node parity).
func interpolatedStacks(raw map[string]any, vars map[string]string) ([]string, error) {
	list, ok := raw["stacks"].([]any)
	if !ok {
		return nil, nil
	}
	stacks := make([]string, 0, len(list))
	for _, item := range list {
		s, err := stringify(item)
		if err != nil {
			return nil, fmt.Errorf("stacks: %w", err)
		}
		s, err = interpolateString(s, vars)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, s)
	}
	return stacks, nil
}

// mapAt returns raw[key] as a map, or an empty map when absent or a different
// shape — mirroring lodash get's forgiving traversal.
func mapAt(raw map[string]any, key string) map[string]any {
	m, _ := raw[key].(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

// deriveParameters maps every declared key to "<path>/<KEY>", sorted for
// deterministic output. Declared keys without a path were a TypeError crash in
// Node; here it is a clear error.
func deriveParameters(section string, path string, keys map[string]string) ([]string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	if path == "" {
		return nil, fmt.Errorf("%s.path must be set when %s keys are declared", section, section)
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, path+"/"+k)
	}
	sort.Strings(names)
	return names, nil
}
