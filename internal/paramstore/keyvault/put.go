package keyvault

import (
	"context"
	"sort"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	"github.com/driverforge/gayle/internal/paramstore"
)

func (s *Store) PutConfigs(ctx context.Context, values map[string]string) ([]paramstore.PutResult, error) {
	return s.put(ctx, values, "config")
}

func (s *Store) PutSecrets(ctx context.Context, values map[string]string) ([]paramstore.PutResult, error) {
	return s.put(ctx, values, "secret")
}

// put prefetches current values, writes only changed keys (tagged with their
// kind so listings can tell config from secret), sequentially in name order,
// attempting every key and aggregating failures.
func (s *Store) put(ctx context.Context, values map[string]string, typeTag string) ([]paramstore.PutResult, error) {
	names := make([]string, 0, len(values))
	for n := range values {
		names = append(names, n)
	}
	sort.Strings(names)

	current, err := s.GetParameters(ctx, names)
	if err != nil {
		return nil, err
	}

	var results []paramstore.PutResult
	var errs paramstore.KeyErrors
	for _, name := range names {
		value := values[name]
		if current[name] == value {
			continue
		}
		tag := typeTag
		res, err := s.client.SetSecret(ctx, ToKeyVaultName(name), azsecrets.SetSecretParameters{
			Value: &value,
			Tags:  map[string]*string{"type": &tag},
		}, nil)
		if err != nil {
			errs = append(errs, paramstore.KeyError{Key: name, Err: err})
			continue
		}
		version := ""
		if res.ID != nil {
			version = res.ID.Version()
		}
		s.cache[name] = value
		results = append(results, paramstore.PutResult{Name: name, Value: value, Version: version})
	}
	return results, errs.OrNil()
}
