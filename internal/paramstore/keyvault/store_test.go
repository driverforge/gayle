package keyvault

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	"github.com/driverforge/gayle/internal/paramstore"
)

func TestKeyNameMangling(t *testing.T) {
	cases := []struct{ internal, kv string }{
		{"graph/DB_NAME", "graph--DB-NAME"},
		{"/dev/config/DB_TABLE", "/dev/config--DB-TABLE"},
	}
	for _, c := range cases {
		if got := ToKeyVaultName(c.internal); got != c.kv {
			t.Errorf("ToKeyVaultName(%q) = %q, want %q", c.internal, got, c.kv)
		}
		if got := FromKeyVaultName(c.kv); got != c.internal {
			t.Errorf("FromKeyVaultName(%q) = %q, want %q", c.kv, got, c.internal)
		}
	}
	// A bare key round-trips asymmetrically ("KEY" → "--KEY" → "/KEY") —
	// pinned against the Node implementation.
	if got := ToKeyVaultName("KEY"); got != "--KEY" {
		t.Errorf("ToKeyVaultName(KEY) = %q, want --KEY", got)
	}
	if got := FromKeyVaultName("--KEY"); got != "/KEY" {
		t.Errorf("FromKeyVaultName(--KEY) = %q, want /KEY (Node parity)", got)
	}
	// No separator → returned as-is.
	if got := FromKeyVaultName("plain"); got != "plain" {
		t.Errorf("FromKeyVaultName(plain) = %q", got)
	}
	// The reverse is lossy: hyphens in the original key come back as underscores.
	if got := FromKeyVaultName(ToKeyVaultName("graph/MY-KEY")); got != "graph/MY_KEY" {
		t.Errorf("lossy round-trip changed: got %q, the Node behavior is graph/MY_KEY", got)
	}
}

func notFoundErr() error {
	return &azcore.ResponseError{
		StatusCode: 404,
		RawResponse: &http.Response{
			StatusCode: 404,
			Request:    &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "fake.vault.azure.net"}},
			Header:     http.Header{},
			Body:       http.NoBody,
		},
	}
}

type fakeSecret struct {
	value   string
	typeTag string
	enabled *bool
}

type fakeAPI struct {
	mu      sync.Mutex
	secrets map[string]fakeSecret // by kv name

	getErr    map[string]error
	setErr    map[string]error
	deleteErr map[string]error
	purgeErr  map[string]error

	deleted      map[string]int // kv name → GetDeletedSecret calls before it reports deleted
	purged       []string
	setCalls     []string
	deleteCalls  []string
	deletedPolls map[string]int
}

func newFake() *fakeAPI {
	return &fakeAPI{
		secrets:      map[string]fakeSecret{},
		getErr:       map[string]error{},
		setErr:       map[string]error{},
		deleteErr:    map[string]error{},
		purgeErr:     map[string]error{},
		deleted:      map[string]int{},
		deletedPolls: map[string]int{},
	}
}

func secretID(name string) *azsecrets.ID {
	id := azsecrets.ID("https://fake.vault.azure.net/secrets/" + name + "/v7")
	return &id
}

func (f *fakeAPI) GetSecret(_ context.Context, name, _ string, _ *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.getErr[name]; err != nil {
		return azsecrets.GetSecretResponse{}, err
	}
	s, ok := f.secrets[name]
	if !ok {
		return azsecrets.GetSecretResponse{}, notFoundErr()
	}
	value := s.value
	tag := s.typeTag
	resp := azsecrets.GetSecretResponse{}
	resp.Value = &value
	resp.ID = secretID(name)
	if tag != "" {
		resp.Tags = map[string]*string{"type": &tag}
	}
	return resp, nil
}

func (f *fakeAPI) SetSecret(_ context.Context, name string, params azsecrets.SetSecretParameters, _ *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setCalls = append(f.setCalls, name)
	if err := f.setErr[name]; err != nil {
		return azsecrets.SetSecretResponse{}, err
	}
	tag := ""
	if params.Tags["type"] != nil {
		tag = *params.Tags["type"]
	}
	f.secrets[name] = fakeSecret{value: *params.Value, typeTag: tag}
	resp := azsecrets.SetSecretResponse{}
	resp.ID = secretID(name)
	return resp, nil
}

func (f *fakeAPI) DeleteSecret(_ context.Context, name string, _ *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls = append(f.deleteCalls, name)
	if err := f.deleteErr[name]; err != nil {
		return azsecrets.DeleteSecretResponse{}, err
	}
	delete(f.secrets, name)
	if _, ok := f.deleted[name]; !ok {
		f.deleted[name] = 0
	}
	return azsecrets.DeleteSecretResponse{}, nil
}

func (f *fakeAPI) GetDeletedSecret(_ context.Context, name string, _ *azsecrets.GetDeletedSecretOptions) (azsecrets.GetDeletedSecretResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	polls, ok := f.deleted[name]
	if !ok {
		return azsecrets.GetDeletedSecretResponse{}, notFoundErr()
	}
	f.deletedPolls[name]++
	if f.deletedPolls[name] <= polls {
		return azsecrets.GetDeletedSecretResponse{}, notFoundErr()
	}
	return azsecrets.GetDeletedSecretResponse{}, nil
}

func (f *fakeAPI) PurgeDeletedSecret(_ context.Context, name string, _ *azsecrets.PurgeDeletedSecretOptions) (azsecrets.PurgeDeletedSecretResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.purgeErr[name]; err != nil {
		return azsecrets.PurgeDeletedSecretResponse{}, err
	}
	f.purged = append(f.purged, name)
	return azsecrets.PurgeDeletedSecretResponse{}, nil
}

func (f *fakeAPI) NewListSecretPropertiesPager(_ *azsecrets.ListSecretPropertiesOptions) *runtime.Pager[azsecrets.ListSecretPropertiesResponse] {
	f.mu.Lock()
	names := make([]string, 0, len(f.secrets))
	for n := range f.secrets {
		names = append(names, n)
	}
	f.mu.Unlock()
	sort.Strings(names)

	// Two items per page to exercise pagination.
	var pages [][]string
	for len(names) > 0 {
		n := min(2, len(names))
		pages = append(pages, names[:n])
		names = names[n:]
	}
	if pages == nil {
		pages = [][]string{{}}
	}

	i := 0
	return runtime.NewPager(runtime.PagingHandler[azsecrets.ListSecretPropertiesResponse]{
		More: func(azsecrets.ListSecretPropertiesResponse) bool { return i < len(pages) },
		Fetcher: func(context.Context, *azsecrets.ListSecretPropertiesResponse) (azsecrets.ListSecretPropertiesResponse, error) {
			page := pages[i]
			i++
			resp := azsecrets.ListSecretPropertiesResponse{}
			f.mu.Lock()
			defer f.mu.Unlock()
			for _, name := range page {
				props := &azsecrets.SecretProperties{ID: secretID(name)}
				if s, ok := f.secrets[name]; ok && s.enabled != nil {
					props.Attributes = &azsecrets.SecretAttributes{Enabled: s.enabled}
				}
				resp.Value = append(resp.Value, props)
			}
			return resp, nil
		},
	})
}

func newTestStore(f *fakeAPI) *Store {
	s := newWithAPI(f)
	s.pollInterval = time.Millisecond
	s.pollTimeout = 100 * time.Millisecond
	return s
}

func TestGetParameters404VsError(t *testing.T) {
	f := newFake()
	f.secrets["graph--DB-NAME"] = fakeSecret{value: "db"}
	s := newTestStore(f)

	got, err := s.GetParameters(context.Background(), []string{"graph/DB_NAME", "graph/MISSING"})
	if err != nil {
		t.Fatal(err)
	}
	if got["graph/DB_NAME"] != "db" || got["graph/MISSING"] != "" {
		t.Errorf("values wrong: %v", got)
	}

	// A non-404 (auth/network) error must fail the read — the Node CLI
	// swallowed it into "".
	f2 := newFake()
	f2.getErr["graph--BROKEN"] = errors.New("401 unauthorized")
	s2 := newTestStore(f2)
	if _, err := s2.GetParameters(context.Background(), []string{"graph/BROKEN"}); err == nil {
		t.Error("non-404 error must fail the read")
	}
}

func TestGetAllByPathFiltersAndTags(t *testing.T) {
	f := newFake()
	off := false
	f.secrets["graph--DB-NAME"] = fakeSecret{value: "db", typeTag: "config"}
	f.secrets["graph--DB-PASSWORD"] = fakeSecret{value: "hunter2", typeTag: "secret"}
	f.secrets["graph--DISABLED"] = fakeSecret{value: "x", typeTag: "config", enabled: &off}
	f.secrets["other--A"] = fakeSecret{value: "y", typeTag: "config"}
	f.secrets["graph--UNTAGGED"] = fakeSecret{value: "z"}
	s := newTestStore(f)

	params, err := s.GetAllByPath(context.Background(), "graph")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]paramstore.Parameter{}
	for _, p := range params {
		byName[p.Name] = p
	}
	if len(params) != 3 {
		t.Fatalf("params = %+v, want 3 (disabled + other-prefix excluded)", params)
	}
	if byName["graph/DB_NAME"].Type != paramstore.TypeString {
		t.Errorf("config tag must map to String: %+v", byName["graph/DB_NAME"])
	}
	if byName["graph/DB_PASSWORD"].Type != paramstore.TypeSecureString {
		t.Errorf("secret tag must map to SecureString")
	}
	// No/unknown tag → SecureString (Node parity: only 'config' is String).
	if byName["graph/UNTAGGED"].Type != paramstore.TypeSecureString {
		t.Errorf("untagged must map to SecureString")
	}
}

func TestPutSkipsUnchangedAndTags(t *testing.T) {
	f := newFake()
	f.secrets["graph--SAME"] = fakeSecret{value: "keep", typeTag: "config"}
	s := newTestStore(f)

	results, err := s.PutConfigs(context.Background(), map[string]string{
		"graph/SAME": "keep",
		"graph/NEW":  "fresh",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "graph/NEW" || results[0].Version != "v7" {
		t.Errorf("results = %+v", results)
	}
	if len(f.setCalls) != 1 {
		t.Errorf("unchanged value must not be written: %v", f.setCalls)
	}
	if f.secrets["graph--NEW"].typeTag != "config" {
		t.Errorf("config write must be tagged type=config")
	}

	if _, err := s.PutSecrets(context.Background(), map[string]string{"graph/P": "x"}); err != nil {
		t.Fatal(err)
	}
	if f.secrets["graph--P"].typeTag != "secret" {
		t.Errorf("secret write must be tagged type=secret")
	}
}

func TestPutAggregatesFailures(t *testing.T) {
	f := newFake()
	f.setErr["graph--BAD"] = errors.New("403 forbidden")
	s := newTestStore(f)
	results, err := s.PutConfigs(context.Background(), map[string]string{
		"graph/BAD": "x",
		"graph/OK":  "y",
	})
	var ke paramstore.KeyErrors
	if !errors.As(err, &ke) || len(ke) != 1 || ke[0].Key != "graph/BAD" {
		t.Fatalf("want 1 KeyError for graph/BAD, got %v", err)
	}
	if len(results) != 1 || results[0].Name != "graph/OK" {
		t.Errorf("other keys must still be attempted: %+v", results)
	}
}

func TestDeletePurgeAsymmetry(t *testing.T) {
	// Delete failure → hard error; purge failure → warning only.
	f := newFake()
	f.secrets["graph--DEL-FAIL"] = fakeSecret{value: "a"}
	f.secrets["graph--PURGE-FAIL"] = fakeSecret{value: "b"}
	f.secrets["graph--OK"] = fakeSecret{value: "c"}
	f.deleteErr["graph--DEL-FAIL"] = errors.New("403 forbidden")
	f.purgeErr["graph--PURGE-FAIL"] = errors.New("purge protection enabled")
	s := newTestStore(f)

	err := s.DeleteParameters(context.Background(), []string{"graph/DEL_FAIL", "graph/PURGE_FAIL", "graph/OK"})
	var ke paramstore.KeyErrors
	if !errors.As(err, &ke) || len(ke) != 1 {
		t.Fatalf("want exactly the delete failure, got %v", err)
	}
	if ke[0].Key != "graph/DEL_FAIL" {
		t.Errorf("failed key = %q", ke[0].Key)
	}
	// Both deletable secrets were deleted and purge attempted on each.
	if len(f.purged) != 1 || f.purged[0] != "graph--OK" {
		t.Errorf("purged = %v, want [graph--OK] (PURGE-FAIL's purge failed but was attempted)", f.purged)
	}
	if _, live := f.secrets["graph--PURGE-FAIL"]; live {
		t.Errorf("PURGE_FAIL should be deleted even though purge failed")
	}
}

func TestDeleteAlreadyGoneIsNotAnError(t *testing.T) {
	// DF-659: a 404 from DeleteSecret means the secret is already absent —
	// pruning's goal state, not a failure.
	f := newFake()
	f.secrets["graph--OK"] = fakeSecret{value: "a"}
	f.deleteErr["graph--GONE"] = notFoundErr()
	s := newTestStore(f)

	if err := s.DeleteParameters(context.Background(), []string{"graph/GONE", "graph/OK"}); err != nil {
		t.Fatalf("already-deleted secret must not fail the batch: %v", err)
	}
	if _, live := f.secrets["graph--OK"]; live {
		t.Errorf("remaining live secret must still be deleted")
	}
}

func TestDeleteWaitsForSoftDelete(t *testing.T) {
	f := newFake()
	f.secrets["graph--SLOW"] = fakeSecret{value: "a"}
	f.deleted["graph--SLOW"] = 3 // three polls before the soft-delete lands
	s := newTestStore(f)

	if err := s.DeleteParameters(context.Background(), []string{"graph/SLOW"}); err != nil {
		t.Fatal(err)
	}
	if f.deletedPolls["graph--SLOW"] < 4 {
		t.Errorf("expected polling until soft-delete confirmed, polls=%d", f.deletedPolls["graph--SLOW"])
	}
	if !strings.Contains(strings.Join(f.purged, ","), "graph--SLOW") {
		t.Errorf("purge must happen after confirmation: %v", f.purged)
	}
}
