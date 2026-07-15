package settings

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/sync/errgroup"

	"github.com/driverforge/gayle/internal/ui"
)

// Region resolves the AWS region the way the Node CLI did: AWS_REGION, then
// AWS_DEFAULT_REGION, then us-east-1.
func Region() string {
	if r := os.Getenv("AWS_REGION"); r != "" {
		return r
	}
	if r := os.Getenv("AWS_DEFAULT_REGION"); r != "" {
		return r
	}
	return "us-east-1"
}

// awsContext gathers the AWS-derived interpolation variables: the caller
// account id, the region, and every CloudFormation output of stackNames
// (later stacks win on OutputKey collisions; accountId/region win over all).
//
// A DescribeStacks failure is a hard error. The Node CLI swallowed it into a
// warning and empty outputs, which then surfaced as a baffling "X is not
// defined" — or worse, silently deployed without the stack's values.
func awsContext(ctx context.Context, stackNames []string) (map[string]string, error) {
	region := Region()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	stsClient := sts.NewFromConfig(cfg)
	cfClient := cloudformation.NewFromConfig(cfg)

	g, gctx := errgroup.WithContext(ctx)

	var accountID string
	g.Go(func() error {
		out, err := stsClient.GetCallerIdentity(gctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return fmt.Errorf("sts get-caller-identity: %w", err)
		}
		if aws.ToString(out.Account) == "" {
			return errors.New("sts get-caller-identity: missing accountId")
		}
		accountID = aws.ToString(out.Account)
		return nil
	})

	// The "Getting stack outputs" banners print in declaration order before any
	// fetch completes, like the Node CLI's synchronous Promise.all setup.
	outputs := make([]map[string]string, len(stackNames))
	for i, name := range stackNames {
		ui.Log(ui.Cyan(fmt.Sprintf("Getting stack outputs for: [%s]", name)))
		if name == "" {
			return nil, errors.New(`Please specify stackName for "stacks"`)
		}
		g.Go(func() error {
			out, err := cfClient.DescribeStacks(gctx, &cloudformation.DescribeStacksInput{StackName: aws.String(name)})
			if err != nil {
				return fmt.Errorf("stack outputs for [%s]: %w", name, err)
			}
			m := map[string]string{}
			if len(out.Stacks) > 0 {
				for _, o := range out.Stacks[0].Outputs {
					m[aws.ToString(o.OutputKey)] = aws.ToString(o.OutputValue)
				}
			}
			outputs[i] = m
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	vars := map[string]string{}
	for _, m := range outputs {
		for k, v := range m {
			vars[k] = v
		}
	}
	vars["accountId"] = accountID
	vars["region"] = region
	return vars, nil
}
