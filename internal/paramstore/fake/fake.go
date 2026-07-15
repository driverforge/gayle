// Package fake provides an in-memory paramstore.Store for command tests.
package fake

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/driverforge/gayle/internal/paramstore"
)

// Store is a map-backed paramstore.Store. Zero value is usable. Not safe for
// concurrent use (command tests are sequential).
type Store struct {
	Values map[string]string               // full name → value
	Types  map[string]paramstore.ParamType // full name → type (default String)

	// Injectable failures.
	GetErr    error            // returned by every read
	PutErr    map[string]error // per-key write failures
	DeleteErr map[string]error // per-key delete failures

	// Recorded activity.
	PutConfigCalls []map[string]string
	PutSecretCalls []map[string]string
	Deleted        []string

	version int
}

func (s *Store) init() {
	if s.Values == nil {
		s.Values = map[string]string{}
	}
	if s.Types == nil {
		s.Types = map[string]paramstore.ParamType{}
	}
}

// Set seeds a parameter.
func (s *Store) Set(name, value string, t paramstore.ParamType) {
	s.init()
	s.Values[name] = value
	s.Types[name] = t
}

func (s *Store) GetParameters(_ context.Context, names []string) (map[string]string, error) {
	s.init()
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	out := make(map[string]string, len(names))
	for _, n := range names {
		out[n] = s.Values[n] // missing → ""
	}
	return out, nil
}

func (s *Store) GetAllByPath(_ context.Context, path string) ([]paramstore.Parameter, error) {
	s.init()
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	var params []paramstore.Parameter
	for name, value := range s.Values {
		if !strings.HasPrefix(name, path+"/") {
			continue
		}
		t := s.Types[name]
		if t == "" {
			t = paramstore.TypeString
		}
		params = append(params, paramstore.Parameter{Name: name, Value: value, Type: t})
	}
	sort.Slice(params, func(i, j int) bool { return params[i].Name < params[j].Name })
	return params, nil
}

func (s *Store) PutConfigs(ctx context.Context, values map[string]string) ([]paramstore.PutResult, error) {
	s.PutConfigCalls = append(s.PutConfigCalls, values)
	return s.put(values, paramstore.TypeString)
}

func (s *Store) PutSecrets(ctx context.Context, values map[string]string) ([]paramstore.PutResult, error) {
	s.PutSecretCalls = append(s.PutSecretCalls, values)
	return s.put(values, paramstore.TypeSecureString)
}

// put mirrors the real stores: sorted order, skip-if-unchanged, attempt every
// key, aggregate failures.
func (s *Store) put(values map[string]string, t paramstore.ParamType) ([]paramstore.PutResult, error) {
	s.init()
	names := make([]string, 0, len(values))
	for n := range values {
		names = append(names, n)
	}
	sort.Strings(names)

	var results []paramstore.PutResult
	var errs paramstore.KeyErrors
	for _, n := range names {
		if err := s.PutErr[n]; err != nil {
			errs = append(errs, paramstore.KeyError{Key: n, Err: err})
			continue
		}
		if existing, ok := s.Values[n]; ok && existing == values[n] {
			continue // unchanged: no write, no result
		}
		s.version++
		s.Values[n] = values[n]
		s.Types[n] = t
		results = append(results, paramstore.PutResult{Name: n, Value: values[n], Version: strconv.Itoa(s.version)})
	}
	return results, errs.OrNil()
}

func (s *Store) DeleteParameters(_ context.Context, names []string) error {
	s.init()
	var errs paramstore.KeyErrors
	for _, n := range names {
		if err := s.DeleteErr[n]; err != nil {
			errs = append(errs, paramstore.KeyError{Key: n, Err: err})
			continue
		}
		delete(s.Values, n)
		delete(s.Types, n)
		s.Deleted = append(s.Deleted, n)
	}
	return errs.OrNil()
}
