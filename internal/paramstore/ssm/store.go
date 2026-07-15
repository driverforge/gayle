// Package ssm implements paramstore.Store on AWS SSM Parameter Store.
package ssm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/driverforge/gayle/internal/settings"
)

// api is the slice of the SSM client the store uses — a seam so tests can
// fake the wire without HTTP mocking. *awsssm.Client satisfies it directly.
type api interface {
	GetParameters(ctx context.Context, in *awsssm.GetParametersInput, opts ...func(*awsssm.Options)) (*awsssm.GetParametersOutput, error)
	GetParametersByPath(ctx context.Context, in *awsssm.GetParametersByPathInput, opts ...func(*awsssm.Options)) (*awsssm.GetParametersByPathOutput, error)
	PutParameter(ctx context.Context, in *awsssm.PutParameterInput, opts ...func(*awsssm.Options)) (*awsssm.PutParameterOutput, error)
	DeleteParameters(ctx context.Context, in *awsssm.DeleteParametersInput, opts ...func(*awsssm.Options)) (*awsssm.DeleteParametersOutput, error)
}

// Store is the SSM-backed paramstore.Store. The read cache plays the role of
// the Node CLI's DataLoader: values read once in a run (e.g. the pre-write
// diff) are never re-fetched.
type Store struct {
	client api
	cache  map[string]string
}

// New builds a Store on the default AWS credential chain, in the region the
// Node CLI resolved (AWS_REGION || AWS_DEFAULT_REGION || us-east-1).
func New(ctx context.Context) (*Store, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(settings.Region()))
	if err != nil {
		return nil, fmt.Errorf("ssm: aws config: %w", err)
	}
	return newWithAPI(awsssm.NewFromConfig(cfg)), nil
}

func newWithAPI(client api) *Store {
	return &Store{client: client, cache: map[string]string{}}
}

// batchSize is SSM's GetParameters/DeleteParameters name limit.
const batchSize = 10

func chunk(names []string) [][]string {
	var chunks [][]string
	for len(names) > 0 {
		n := min(batchSize, len(names))
		chunks = append(chunks, names[:n])
		names = names[n:]
	}
	return chunks
}
