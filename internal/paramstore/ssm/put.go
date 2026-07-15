package ssm

import (
	"context"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/driverforge/gayle/internal/paramstore"
)

// secretKeyID is the KMS key SSM secrets are encrypted with. The Node CLI
// hard-coded this (a yml `secret.keyId` was documented but never honored);
// changing it would silently re-encrypt existing parameters, so it stays.
const secretKeyID = "alias/aws/ssm"

func (s *Store) PutConfigs(ctx context.Context, values map[string]string) ([]paramstore.PutResult, error) {
	return s.put(ctx, values, types.ParameterTypeString)
}

func (s *Store) PutSecrets(ctx context.Context, values map[string]string) ([]paramstore.PutResult, error) {
	return s.put(ctx, values, types.ParameterTypeSecureString)
}

// put prefetches the current values in one batched read, then writes only the
// keys whose value changed (no version churn — Node parity). Writes run
// sequentially in name order for deterministic logs; every key is attempted
// and failures are aggregated so a partial failure reports the full damage.
func (s *Store) put(ctx context.Context, values map[string]string, paramType types.ParameterType) ([]paramstore.PutResult, error) {
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
		in := &awsssm.PutParameterInput{
			Name:      aws.String(name),
			Type:      paramType,
			Value:     aws.String(value),
			Overwrite: aws.Bool(true),
		}
		if paramType == types.ParameterTypeSecureString {
			in.KeyId = aws.String(secretKeyID)
		}
		res, err := s.client.PutParameter(ctx, in)
		if err != nil {
			errs = append(errs, paramstore.KeyError{Key: name, Err: err})
			continue
		}
		s.cache[name] = value
		results = append(results, paramstore.PutResult{
			Name:    name,
			Value:   value,
			Version: strconv.FormatInt(res.Version, 10),
		})
	}
	return results, errs.OrNil()
}
