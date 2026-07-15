package settings

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type fakeSTS struct {
	account string
	err     error
}

func (f *fakeSTS) GetCallerIdentity(context.Context, *sts.GetCallerIdentityInput, ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := &sts.GetCallerIdentityOutput{}
	if f.account != "" {
		out.Account = aws.String(f.account)
	}
	return out, nil
}

type fakeCF struct {
	outputs map[string]map[string]string // stack → OutputKey → OutputValue
	errs    map[string]error
}

func (f *fakeCF) DescribeStacks(_ context.Context, in *cloudformation.DescribeStacksInput, _ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	name := aws.ToString(in.StackName)
	if err := f.errs[name]; err != nil {
		return nil, err
	}
	var outs []cftypes.Output
	for k, v := range f.outputs[name] {
		outs = append(outs, cftypes.Output{OutputKey: aws.String(k), OutputValue: aws.String(v)})
	}
	return &cloudformation.DescribeStacksOutput{Stacks: []cftypes.Stack{{Outputs: outs}}}, nil
}

func TestAWSContextMergesOutputsAndIdentity(t *testing.T) {
	stsc := &fakeSTS{account: "123456789012"}
	cfc := &fakeCF{outputs: map[string]map[string]string{
		"stack-a": {"UserPoolId": "pool-a", "Shared": "from-a"},
		"stack-b": {"Shared": "from-b", "accountId": "output-tries-to-win"},
	}}
	vars, err := awsContextWith(context.Background(), stsc, cfc, "ap-southeast-2", []string{"stack-a", "stack-b"})
	if err != nil {
		t.Fatal(err)
	}
	if vars["UserPoolId"] != "pool-a" {
		t.Errorf("missing stack output: %v", vars)
	}
	// Later stacks win on collisions; accountId/region win over any output.
	if vars["Shared"] != "from-b" {
		t.Errorf("stack merge order wrong: %v", vars)
	}
	if vars["accountId"] != "123456789012" || vars["region"] != "ap-southeast-2" {
		t.Errorf("identity must win over outputs: %v", vars)
	}
}

// A DescribeStacks failure is a hard error — the Node CLI swallowed it into a
// warning and empty outputs.
func TestAWSContextStackFailureIsHard(t *testing.T) {
	stsc := &fakeSTS{account: "123456789012"}
	cfc := &fakeCF{errs: map[string]error{"missing-stack": errors.New("Stack with id missing-stack does not exist")}}
	_, err := awsContextWith(context.Background(), stsc, cfc, "us-east-1", []string{"missing-stack"})
	if err == nil || !strings.Contains(err.Error(), "stack outputs for [missing-stack]") {
		t.Fatalf("DescribeStacks failure must be a hard error naming the stack: %v", err)
	}
}

func TestAWSContextIdentityErrors(t *testing.T) {
	if _, err := awsContextWith(context.Background(), &fakeSTS{err: errors.New("ExpiredToken")}, &fakeCF{}, "us-east-1", nil); err == nil || !strings.Contains(err.Error(), "sts get-caller-identity") {
		t.Errorf("sts failure must propagate: %v", err)
	}
	if _, err := awsContextWith(context.Background(), &fakeSTS{}, &fakeCF{}, "us-east-1", nil); err == nil || !strings.Contains(err.Error(), "missing accountId") {
		t.Errorf("empty account must be an error: %v", err)
	}
}

func TestAWSContextEmptyStackName(t *testing.T) {
	_, err := awsContextWith(context.Background(), &fakeSTS{account: "1"}, &fakeCF{}, "us-east-1", []string{""})
	if err == nil || !strings.Contains(err.Error(), `Please specify stackName for "stacks"`) {
		t.Errorf("empty stack name must error: %v", err)
	}
}

func TestRegionResolution(t *testing.T) {
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	if got := Region(); got != "us-east-1" {
		t.Errorf("default region = %q", got)
	}
	t.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
	if got := Region(); got != "eu-west-1" {
		t.Errorf("AWS_DEFAULT_REGION not honoured: %q", got)
	}
	t.Setenv("AWS_REGION", "ap-southeast-2")
	if got := Region(); got != "ap-southeast-2" {
		t.Errorf("AWS_REGION must win: %q", got)
	}
}
