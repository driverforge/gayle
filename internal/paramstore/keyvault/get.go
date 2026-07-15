package keyvault

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/driverforge/gayle/internal/paramstore"
)

// getConcurrency bounds parallel per-secret reads (Key Vault has no batch API).
const getConcurrency = 8

// GetParameters reads each name (mangled to its vault secret name). A 404
// maps to "" — the missing-required flow depends on that; any other error
// (auth, network, throttle) fails the read.
func (s *Store) GetParameters(ctx context.Context, names []string) (map[string]string, error) {
	out := make(map[string]string, len(names))
	var toFetch []string
	for _, n := range names {
		if v, ok := s.cache[n]; ok {
			out[n] = v
		} else {
			toFetch = append(toFetch, n)
		}
	}
	if len(toFetch) == 0 {
		return out, nil
	}

	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(getConcurrency)
	for _, name := range toFetch {
		g.Go(func() error {
			res, err := s.client.GetSecret(gctx, ToKeyVaultName(name), "", nil)
			value := ""
			switch {
			case err == nil:
				if res.Value != nil {
					value = *res.Value
				}
			case isNotFound(err):
				// missing → ""
			default:
				return fmt.Errorf("key vault get %s: %w", name, err)
			}
			mu.Lock()
			out[name] = value
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	for _, n := range toFetch {
		s.cache[n] = out[n]
	}
	return out, nil
}

// GetAllByPath lists the whole vault (Key Vault has no prefix query), keeps
// enabled secrets whose name starts with "<path>--", reads each one, and maps
// the type tag onto String/SecureString. Results are in vault listing order
// filtered, matching the Node CLI's sequential loop, but reads are bounded-
// concurrent.
func (s *Store) GetAllByPath(ctx context.Context, path string) ([]paramstore.Parameter, error) {
	prefix := path + separator

	var matches []string
	pager := s.client.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("key vault list: %w", err)
		}
		for _, item := range page.Value {
			if item.ID == nil {
				continue
			}
			name := item.ID.Name()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			if item.Attributes != nil && item.Attributes.Enabled != nil && !*item.Attributes.Enabled {
				continue
			}
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)

	params := make([]paramstore.Parameter, len(matches))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(getConcurrency)
	for i, kvName := range matches {
		g.Go(func() error {
			res, err := s.client.GetSecret(gctx, kvName, "", nil)
			if err != nil {
				return fmt.Errorf("key vault get %s: %w", kvName, err)
			}
			value := ""
			if res.Value != nil {
				value = *res.Value
			}
			paramType := paramstore.TypeSecureString
			if tag, ok := res.Tags["type"]; ok && tag != nil && *tag == "config" {
				paramType = paramstore.TypeString
			}
			params[i] = paramstore.Parameter{
				Name:  FromKeyVaultName(kvName),
				Value: value,
				Type:  paramType,
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return params, nil
}
