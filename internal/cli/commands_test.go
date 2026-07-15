package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/driverforge/gayle/internal/clierr"
	"github.com/driverforge/gayle/internal/paramstore"
	"github.com/driverforge/gayle/internal/paramstore/fake"
	"github.com/driverforge/gayle/internal/settings"
)

// fixtureSettings is the interpolated shape of a typical ssm gayle.yml at
// stage dev: two defaults, one required config, one required secret.
func fixtureSettings() *settings.Settings {
	return &settings.Settings{
		Service:  "my-service",
		Provider: settings.Provider{Name: "ssm"},
		Config: &settings.ConfigBlock{
			Path:     "/dev/config",
			Defaults: map[string]string{"DB_NAME": "my-database", "DB_HOST": "3200"},
			Required: map[string]string{"DB_TABLE": "some table"},
		},
		Secret: &settings.SecretBlock{
			Path:     "/dev/secret",
			Required: map[string]string{"DB_PASSWORD": "the password"},
		},
		ConfigParameters: []string{"/dev/config/DB_HOST", "/dev/config/DB_NAME", "/dev/config/DB_TABLE"},
		SecretParameters: []string{"/dev/secret/DB_PASSWORD"},
	}
}

func testDeps(s *settings.Settings, store paramstore.Store) *deps {
	return &deps{
		load: func(context.Context, string, map[string]string, string) (*settings.Settings, error) {
			return s, nil
		},
		newStore: func(context.Context, *settings.Settings) (paramstore.Store, error) {
			return store, nil
		},
	}
}

func run(t *testing.T, d *deps, args ...string) (stdout string, err error) {
	t.Helper()
	resetFlags()
	root := newRootCmd(d)
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(args)
	cmd, err := root.ExecuteC()
	return out.String(), friendlyUsage(cmd, err)
}

// seed populates the fake with every declared parameter's remote value.
func seed(st *fake.Store) {
	st.Set("/dev/config/DB_NAME", "my-database", paramstore.TypeString)
	st.Set("/dev/config/DB_HOST", "3200", paramstore.TypeString)
	st.Set("/dev/config/DB_TABLE", "the-table", paramstore.TypeString)
	st.Set("/dev/secret/DB_PASSWORD", "hunter2hunter2", paramstore.TypeSecureString)
}

func TestRunNonInteractiveMissingRequired(t *testing.T) {
	st := &fake.Store{}
	// Defaults exist remotely but the required DB_TABLE and DB_PASSWORD don't.
	st.Set("/dev/config/DB_NAME", "my-database", paramstore.TypeString)
	_, err := run(t, testDeps(fixtureSettings(), st), "run", "-s", "dev")
	if err == nil || !strings.Contains(err.Error(), "Missing required configs!! Run on interactive mode to populate them!!") {
		t.Fatalf("want missing-required error, got %v", err)
	}
	if !clierr.IsUser(err) {
		t.Errorf("must be a UserError (exit 1)")
	}
	// Nothing may have been written: the failure aborts before updateConfigs.
	if len(st.PutConfigCalls) != 0 || len(st.PutSecretCalls) != 0 {
		t.Errorf("run must not write when required values are missing")
	}
}

func TestRunNonInteractiveHappyPath(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	// Change a default remotely so the run has something to write back.
	st.Set("/dev/config/DB_NAME", "drifted", paramstore.TypeString)

	_, err := run(t, testDeps(fixtureSettings(), st), "run", "-s", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if st.Values["/dev/config/DB_NAME"] != "my-database" {
		t.Errorf("default not written back: %q", st.Values["/dev/config/DB_NAME"])
	}
	// Required keys keep their remote values (validated, not re-invented).
	if st.Values["/dev/config/DB_TABLE"] != "the-table" {
		t.Errorf("required config overwritten: %q", st.Values["/dev/config/DB_TABLE"])
	}
	if len(st.Deleted) != 0 {
		t.Errorf("run without -r must not clean up")
	}
}

func TestRunPartialWriteFailureExitsNonZero(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	st.Set("/dev/config/DB_NAME", "drifted", paramstore.TypeString)
	st.Set("/dev/config/DB_HOST", "drifted-too", paramstore.TypeString)
	st.PutErr = map[string]error{"/dev/config/DB_HOST": errors.New("AccessDenied")}

	_, err := run(t, testDeps(fixtureSettings(), st), "run", "-s", "dev")
	if err == nil || !strings.Contains(err.Error(), "/dev/config/DB_HOST") {
		t.Fatalf("partial failure must fail naming the key, got %v", err)
	}
	// The other key was still attempted (written) before the run failed.
	if st.Values["/dev/config/DB_NAME"] != "my-database" {
		t.Errorf("non-failing keys must still be attempted")
	}
}

func TestRunStageOverridesWin(t *testing.T) {
	s := fixtureSettings()
	s.Config.StageOverrides = map[string]string{"DB_NAME": "prod-db"}
	st := &fake.Store{}
	seed(st)
	_, err := run(t, testDeps(s, st), "run", "-s", "production")
	if err != nil {
		t.Fatal(err)
	}
	if st.Values["/dev/config/DB_NAME"] != "prod-db" {
		t.Errorf("stage override must win over default: %q", st.Values["/dev/config/DB_NAME"])
	}
}

func TestRunInteractiveWithoutTTYFailsFast(t *testing.T) {
	t.Setenv("CI", "1")
	st := &fake.Store{}
	seed(st)
	_, err := run(t, testDeps(fixtureSettings(), st), "run", "-s", "dev", "-i")
	if err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("interactive mode without a TTY must fail fast, got %v", err)
	}
}

func TestRunRemovingCleansUp(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	st.Set("/dev/config/ORPHAN", "old", paramstore.TypeString)
	_, err := run(t, testDeps(fixtureSettings(), st), "run", "-s", "dev", "-r")
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Deleted) != 1 || st.Deleted[0] != "/dev/config/ORPHAN" {
		t.Errorf("run -r must delete orphans: %v", st.Deleted)
	}
}

func TestRunInvalidVariablesJSON(t *testing.T) {
	st := &fake.Store{}
	_, err := run(t, testDeps(fixtureSettings(), st), "run", "-s", "dev", "-v", "{not json")
	if err == nil || !strings.Contains(err.Error(), "Variables must be in JSON format!!") {
		t.Fatalf("want JSON format error, got %v", err)
	}
}

func TestFetchOutputsJSONOnStdout(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	out, err := run(t, testDeps(fixtureSettings(), st), "fetch", "-s", "dev", "-k", "DB_NAME,DB_PASSWORD")
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("stdout is not clean JSON: %q", out)
	}
	if parsed["DB_NAME"] != "my-database" || parsed["DB_PASSWORD"] != "hunter2hunter2" {
		t.Errorf("fetch values wrong: %v", parsed)
	}
}

func TestFetchUnknownKeyErrors(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	_, err := run(t, testDeps(fixtureSettings(), st), "fetch", "-s", "dev", "-k", "DB_NAME,NOPE")
	if err == nil || !strings.Contains(err.Error(), "NOPE") {
		t.Fatalf("undeclared key must error naming it, got %v", err)
	}
}

func TestFetchWithoutKeysIsUsageError(t *testing.T) {
	st := &fake.Store{}
	_, err := run(t, testDeps(fixtureSettings(), st), "fetch", "-s", "dev")
	if err == nil || !clierr.IsUser(err) {
		t.Fatalf("missing -k must be a usage error, got %v", err)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	path := filepath.Join(t.TempDir(), "exports.json")

	if _, err := run(t, testDeps(fixtureSettings(), st), "export", "-s", "dev", "-p", path); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Configs map[string]string `json:"configs"`
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatalf("export not parseable: %v", err)
	}
	if payload.Configs["DB_NAME"] != "my-database" || payload.Secrets["DB_PASSWORD"] != "hunter2hunter2" {
		t.Errorf("export payload wrong: %+v", payload)
	}

	// Import the same file into an empty store.
	st2 := &fake.Store{}
	if _, err := run(t, testDeps(fixtureSettings(), st2), "import", "-s", "dev", "-p", path); err != nil {
		t.Fatal(err)
	}
	if st2.Values["/dev/config/DB_NAME"] != "my-database" || st2.Values["/dev/secret/DB_PASSWORD"] != "hunter2hunter2" {
		t.Errorf("import did not write values: %v", st2.Values)
	}
}

func TestExportEnvFormat(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	path := filepath.Join(t.TempDir(), ".env_gayle")
	if _, err := run(t, testDeps(fixtureSettings(), st), "export", "-s", "dev", "-t", "env", "-p", path); err != nil {
		t.Fatal(err)
	}
	contents, _ := os.ReadFile(path)
	got := string(contents)
	if !strings.Contains(got, "DB_NAME=\"my-database\"\n") {
		t.Errorf("env format wrong:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("env export must end with a newline")
	}
}

func TestExportConfigOnly(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	path := filepath.Join(t.TempDir(), "exports.json")
	if _, err := run(t, testDeps(fixtureSettings(), st), "export", "-s", "dev", "-p", path, "-C"); err != nil {
		t.Fatal(err)
	}
	contents, _ := os.ReadFile(path)
	if !strings.Contains(string(contents), `"secrets": {}`) {
		t.Errorf("config-only export must have empty secrets: %s", contents)
	}
}

func TestExportInvalidTarget(t *testing.T) {
	st := &fake.Store{}
	_, err := run(t, testDeps(fixtureSettings(), st), "export", "-s", "dev", "-t", "xml")
	if err == nil || !strings.Contains(err.Error(), "json|env") {
		t.Fatalf("invalid target must be a usage error, got %v", err)
	}
}

func TestImportEmptySectionsIsNoOp(t *testing.T) {
	// The Node CLI crashed on {"configs":{},"secrets":{}}.
	st := &fake.Store{}
	path := filepath.Join(t.TempDir(), "empty.json")
	os.WriteFile(path, []byte(`{"configs":{},"secrets":{}}`), 0o644)
	if _, err := run(t, testDeps(fixtureSettings(), st), "import", "-s", "dev", "-p", path); err != nil {
		t.Fatalf("empty import must succeed: %v", err)
	}
	if len(st.PutConfigCalls)+len(st.PutSecretCalls) != 0 {
		t.Errorf("empty import must not write")
	}
}

func TestCleanUpDryRunDeletesNothing(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	st.Set("/dev/config/ORPHAN", "old", paramstore.TypeString)
	if _, err := run(t, testDeps(fixtureSettings(), st), "clean-up", "-s", "dev", "-d"); err != nil {
		t.Fatal(err)
	}
	if len(st.Deleted) != 0 {
		t.Errorf("dry run must not delete: %v", st.Deleted)
	}
}

func TestCleanUpDeletesOrphansOnly(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	st.Set("/dev/config/ORPHAN", "old", paramstore.TypeString)
	st.Set("/dev/secret/OLD_KEY", "x", paramstore.TypeSecureString)
	if _, err := run(t, testDeps(fixtureSettings(), st), "clean-up", "-s", "dev"); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"/dev/config/ORPHAN": true, "/dev/secret/OLD_KEY": true}
	if len(st.Deleted) != 2 || !want[st.Deleted[0]] || !want[st.Deleted[1]] {
		t.Errorf("deleted = %v, want exactly the orphans", st.Deleted)
	}
	if _, ok := st.Values["/dev/config/DB_NAME"]; !ok {
		t.Errorf("declared parameters must survive cleanup")
	}
}

func TestCleanUpRefusesEmptyDeclaration(t *testing.T) {
	s := fixtureSettings()
	s.ConfigParameters = nil
	s.SecretParameters = nil
	st := &fake.Store{}
	seed(st)
	_, err := run(t, testDeps(s, st), "clean-up", "-s", "dev")
	if err == nil || !strings.Contains(err.Error(), "Cleanup refused") {
		t.Fatalf("DF-644 guard missing: %v", err)
	}
	if len(st.Deleted) != 0 {
		t.Errorf("refused cleanup must not delete")
	}
}

func TestCleanUpDeleteFailureExitsNonZero(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	st.Set("/dev/config/ORPHAN", "old", paramstore.TypeString)
	st.DeleteErr = map[string]error{"/dev/config/ORPHAN": errors.New("403 forbidden")}
	_, err := run(t, testDeps(fixtureSettings(), st), "clean-up", "-s", "dev")
	if err == nil || !strings.Contains(err.Error(), "/dev/config/ORPHAN") {
		t.Fatalf("delete failure must fail naming the key: %v", err)
	}
}

func TestGenerateWritesTemplateAndRefusesSecond(t *testing.T) {
	t.Chdir(t.TempDir())
	st := &fake.Store{}
	if _, err := run(t, testDeps(fixtureSettings(), st), "generate"); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile("gayle.yml")
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != settings.GenerateTemplate {
		t.Errorf("template mismatch:\n%s", contents)
	}
	_, err = run(t, testDeps(fixtureSettings(), st), "generate")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second generate must refuse: %v", err)
	}
}

func TestListSucceeds(t *testing.T) {
	st := &fake.Store{}
	seed(st)
	if _, err := run(t, testDeps(fixtureSettings(), st), "list", "-s", "dev"); err != nil {
		t.Fatal(err)
	}
}

func TestInitValidatesSettings(t *testing.T) {
	st := &fake.Store{}
	if _, err := run(t, testDeps(fixtureSettings(), st), "init", "-s", "dev"); err != nil {
		t.Fatal(err)
	}
	d := testDeps(nil, st)
	d.load = func(context.Context, string, map[string]string, string) (*settings.Settings, error) {
		return nil, errors.New("Could not find gayle.yml in the following directory - /x")
	}
	_, err := run(t, d, "init", "-s", "dev")
	if err == nil || !clierr.IsUser(err) {
		t.Fatalf("settings failure must be a UserError: %v", err)
	}
}
