package ssm

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	"golang.org/x/sync/errgroup"

	"github.com/driverforge/gayle/internal/paramstore"
)

// getConcurrency bounds parallel GetParameters chunks. The Node CLI fired all
// chunks at once; a small bound keeps large sets clear of SSM throttling.
const getConcurrency = 4

// GetParameters batch-reads names (decrypted), 10 per call. A parameter SSM
// reports in InvalidParameters (i.e. it does not exist) maps to "" — that
// emptiness drives the missing-required flow. Any API error fails the read.
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
	for _, names := range chunk(toFetch) {
		g.Go(func() error {
			res, err := s.client.GetParameters(gctx, &awsssm.GetParametersInput{
				Names:          names,
				WithDecryption: aws.Bool(true),
			})
			if err != nil {
				return fmt.Errorf("ssm get-parameters: %w", err)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, name := range names {
				out[name] = "" // InvalidParameters (missing) → empty
			}
			for _, p := range res.Parameters {
				out[aws.ToString(p.Name)] = aws.ToString(p.Value)
			}
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

// GetAllByPath returns every parameter under path, recursively, decrypted,
// following NextToken pagination.
func (s *Store) GetAllByPath(ctx context.Context, path string) ([]paramstore.Parameter, error) {
	var params []paramstore.Parameter
	var nextToken *string
	for {
		res, err := s.client.GetParametersByPath(ctx, &awsssm.GetParametersByPathInput{
			Path:           aws.String(path),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
			NextToken:      nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ssm get-parameters-by-path %s: %w", path, err)
		}
		for _, p := range res.Parameters {
			params = append(params, paramstore.Parameter{
				Name:  aws.ToString(p.Name),
				Value: aws.ToString(p.Value),
				Type:  paramstore.ParamType(p.Type),
			})
		}
		if res.NextToken == nil {
			return params, nil
		}
		nextToken = res.NextToken
	}
}
