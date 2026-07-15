package ssm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/driverforge/gayle/internal/paramstore"
)

// fakeAPI implements the api seam in memory.
type fakeAPI struct {
	mu     sync.Mutex
	values map[string]string // existing parameters
	types  map[string]types.ParameterType

	getCalls    [][]string
	putCalls    []*awsssm.PutParameterInput
	deleteCalls [][]string

	putErr map[string]error
	pages  [][]types.Parameter // for GetParametersByPath pagination
}

func (f *fakeAPI) GetParameters(_ context.Context, in *awsssm.GetParametersInput, _ ...func(*awsssm.Options)) (*awsssm.GetParametersOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(in.Names) > 10 {
		return nil, fmt.Errorf("too many names in one call: %d", len(in.Names))
	}
	if in.WithDecryption == nil || !*in.WithDecryption {
		return nil, errors.New("WithDecryption not set")
	}
	f.getCalls = append(f.getCalls, in.Names)
	out := &awsssm.GetParametersOutput{}
	for _, name := range in.Names {
		if v, ok := f.values[name]; ok {
			out.Parameters = append(out.Parameters, types.Parameter{Name: aws.String(name), Value: aws.String(v)})
		} else {
			out.InvalidParameters = append(out.InvalidParameters, name)
		}
	}
	return out, nil
}

func (f *fakeAPI) GetParametersByPath(_ context.Context, in *awsssm.GetParametersByPathInput, _ ...func(*awsssm.Options)) (*awsssm.GetParametersByPathOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	page := 0
	if in.NextToken != nil {
		fmt.Sscanf(*in.NextToken, "page-%d", &page)
	}
	out := &awsssm.GetParametersByPathOutput{Parameters: f.pages[page]}
	if page+1 < len(f.pages) {
		out.NextToken = aws.String(fmt.Sprintf("page-%d", page+1))
	}
	return out, nil
}

func (f *fakeAPI) PutParameter(_ context.Context, in *awsssm.PutParameterInput, _ ...func(*awsssm.Options)) (*awsssm.PutParameterOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putCalls = append(f.putCalls, in)
	if err := f.putErr[aws.ToString(in.Name)]; err != nil {
		return nil, err
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[aws.ToString(in.Name)] = aws.ToString(in.Value)
	return &awsssm.PutParameterOutput{Version: 2}, nil
}

func (f *fakeAPI) DeleteParameters(_ context.Context, in *awsssm.DeleteParametersInput, _ ...func(*awsssm.Options)) (*awsssm.DeleteParametersOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls = append(f.deleteCalls, in.Names)
	out := &awsssm.DeleteParametersOutput{}
	for _, name := range in.Names {
		if _, ok := f.values[name]; ok {
			delete(f.values, name)
			out.DeletedParameters = append(out.DeletedParameters, name)
		} else {
			out.InvalidParameters = append(out.InvalidParameters, name)
		}
	}
	return out, nil
}

func TestGetParametersChunksAndMissing(t *testing.T) {
	f := &fakeAPI{values: map[string]string{}}
	var names []string
	for i := 0; i < 23; i++ {
		name := fmt.Sprintf("/dev/config/KEY_%02d", i)
		names = append(names, name)
		if i%2 == 0 {
			f.values[name] = fmt.Sprintf("v%d", i)
		}
	}
	s := newWithAPI(f)
	got, err := s.GetParameters(context.Background(), names)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.getCalls) != 3 {
		t.Errorf("chunks = %d, want 3 (23 names / 10)", len(f.getCalls))
	}
	if got["/dev/config/KEY_02"] != "v2" || got["/dev/config/KEY_01"] != "" {
		t.Errorf("value mapping wrong: %v", got)
	}

	// Second read must hit the cache, not the API (DataLoader parity).
	calls := len(f.getCalls)
	if _, err := s.GetParameters(context.Background(), names[:5]); err != nil {
		t.Fatal(err)
	}
	if len(f.getCalls) != calls {
		t.Errorf("cached read still hit the API")
	}
}

func TestGetAllByPathPaginates(t *testing.T) {
	f := &fakeAPI{pages: [][]types.Parameter{
		{{Name: aws.String("/dev/config/A"), Value: aws.String("1"), Type: types.ParameterTypeString}},
		{{Name: aws.String("/dev/secret/B"), Value: aws.String("2"), Type: types.ParameterTypeSecureString}},
	}}
	s := newWithAPI(f)
	params, err := s.GetAllByPath(context.Background(), "/dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(params) != 2 || params[1].Type != paramstore.TypeSecureString {
		t.Errorf("pagination/types wrong: %+v", params)
	}
}

func TestPutSkipsUnchangedAndSetsWireFields(t *testing.T) {
	f := &fakeAPI{values: map[string]string{"/dev/config/SAME": "keep", "/dev/config/CHANGED": "old"}}
	s := newWithAPI(f)
	results, err := s.PutConfigs(context.Background(), map[string]string{
		"/dev/config/SAME":    "keep",
		"/dev/config/CHANGED": "new",
		"/dev/config/NEW":     "fresh",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v, want 2 writes (SAME skipped)", results)
	}
	if len(f.putCalls) != 2 {
		t.Fatalf("putCalls = %d, want 2", len(f.putCalls))
	}
	for _, in := range f.putCalls {
		if in.Type != types.ParameterTypeString || !aws.ToBool(in.Overwrite) || in.KeyId != nil {
			t.Errorf("config put wire fields wrong: %+v", in)
		}
	}
	if results[0].Version != "2" {
		t.Errorf("version not surfaced: %+v", results[0])
	}
}

func TestPutSecretsUsesKMSAlias(t *testing.T) {
	f := &fakeAPI{}
	s := newWithAPI(f)
	if _, err := s.PutSecrets(context.Background(), map[string]string{"/dev/secret/P": "x"}); err != nil {
		t.Fatal(err)
	}
	in := f.putCalls[0]
	if in.Type != types.ParameterTypeSecureString || aws.ToString(in.KeyId) != "alias/aws/ssm" {
		t.Errorf("secret put wire fields wrong: %+v", in)
	}
}

func TestPutAggregatesPerKeyFailures(t *testing.T) {
	f := &fakeAPI{putErr: map[string]error{
		"/dev/config/BAD1": errors.New("AccessDenied"),
		"/dev/config/BAD2": errors.New("Throttled"),
	}}
	s := newWithAPI(f)
	results, err := s.PutConfigs(context.Background(), map[string]string{
		"/dev/config/BAD1": "x",
		"/dev/config/BAD2": "y",
		"/dev/config/OK":   "z",
	})
	if err == nil {
		t.Fatal("want aggregated error")
	}
	var ke paramstore.KeyErrors
	if !errors.As(err, &ke) || len(ke) != 2 {
		t.Fatalf("want 2 KeyErrors, got %v", err)
	}
	// Every key must still have been attempted; OK succeeded.
	if len(results) != 1 || results[0].Name != "/dev/config/OK" {
		t.Errorf("successful write not reported: %+v", results)
	}
	if !strings.Contains(err.Error(), "/dev/config/BAD1") || !strings.Contains(err.Error(), "/dev/config/BAD2") {
		t.Errorf("error must name every failed key: %v", err)
	}
}

func TestDeleteChunksAndReportsInvalid(t *testing.T) {
	f := &fakeAPI{values: map[string]string{}}
	var names []string
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("/dev/config/D%02d", i)
		names = append(names, name)
		if i != 3 {
			f.values[name] = "v"
		}
	}
	s := newWithAPI(f)
	err := s.DeleteParameters(context.Background(), names)
	if len(f.deleteCalls) != 2 {
		t.Errorf("delete chunks = %d, want 2", len(f.deleteCalls))
	}
	// D03 didn't exist → InvalidParameters → reported, not swallowed.
	if err == nil || !strings.Contains(err.Error(), "/dev/config/D03") {
		t.Errorf("invalid delete must be reported: %v", err)
	}
}
