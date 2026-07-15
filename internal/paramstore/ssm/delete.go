package ssm

import (
	"context"
	"errors"

	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/driverforge/gayle/internal/paramstore"
)

// DeleteParameters deletes names 10 per call, sequentially. Every chunk is
// attempted; an API failure marks that chunk's names failed, and any name SSM
// echoes back in InvalidParameters is reported too (the Node CLI ignored
// that response — a delete could no-op and still "succeed").
func (s *Store) DeleteParameters(ctx context.Context, names []string) error {
	var errs paramstore.KeyErrors
	for _, batch := range chunk(names) {
		res, err := s.client.DeleteParameters(ctx, &awsssm.DeleteParametersInput{Names: batch})
		if err != nil {
			for _, name := range batch {
				errs = append(errs, paramstore.KeyError{Key: name, Err: err})
			}
			continue
		}
		for _, name := range res.InvalidParameters {
			errs = append(errs, paramstore.KeyError{Key: name, Err: errors.New("not deleted: invalid parameter")})
		}
		for _, name := range res.DeletedParameters {
			delete(s.cache, name)
		}
	}
	return errs.OrNil()
}
