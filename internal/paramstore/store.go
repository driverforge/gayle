// Package paramstore defines the provider-neutral parameter store contract
// implemented by the ssm and keyvault subpackages, plus the shared per-key
// failure reporting that keeps gayle's exit codes honest.
package paramstore

import (
	"context"
	"strings"
)

// ParamType mirrors SSM's parameter types; the Key Vault store maps its
// `type` tag onto the same two values so masking works identically.
type ParamType string

const (
	TypeString       ParamType = "String"
	TypeSecureString ParamType = "SecureString"
)

// Parameter is a remote parameter as returned by GetAllByPath.
type Parameter struct {
	Name  string
	Value string
	Type  ParamType
}

// PutResult describes one parameter that was actually written (values already
// matching the remote store are skipped and produce no result — the Node
// CLI's "don't churn versions" behavior).
type PutResult struct {
	Name    string
	Value   string
	Version string
}

// Store abstracts SSM / Key Vault. Names are full parameter paths (e.g.
// "/dev/config/DB_HOST"); the Key Vault implementation mangles them to valid
// secret names internally.
//
// Honesty contract: reads fail on any genuine API/auth/transport error — only
// a definitively missing parameter maps to "". Writes and deletes attempt
// every key, then report all failures via KeyErrors; they return nil only
// when every key verifiably succeeded.
type Store interface {
	// GetParameters batch-reads names, decrypted. Missing parameters map to "".
	GetParameters(ctx context.Context, names []string) (map[string]string, error)

	// GetAllByPath returns every parameter under path (recursive, decrypted).
	GetAllByPath(ctx context.Context, path string) ([]Parameter, error)

	// PutConfigs / PutSecrets write name→value as String / SecureString
	// respectively, skipping values that already match the remote store.
	// Results describe the writes that happened, in name order.
	PutConfigs(ctx context.Context, values map[string]string) ([]PutResult, error)
	PutSecrets(ctx context.Context, values map[string]string) ([]PutResult, error)

	// DeleteParameters deletes by full name; any name not verifiably deleted
	// is reported in the returned KeyErrors.
	DeleteParameters(ctx context.Context, names []string) error
}

// ShortKey is the last '/'-segment of a full parameter name — the key the
// yml declares and every human-facing surface (list, fetch, export) shows.
func ShortKey(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// ShortKeys re-keys a full-name→value map by short key, dropping entries
// whose short key is empty (Node parity: a trailing-slash name vanishes).
func ShortKeys(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for name, value := range params {
		if key := ShortKey(name); key != "" {
			out[key] = value
		}
	}
	return out
}
