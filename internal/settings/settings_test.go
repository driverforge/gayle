package settings

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeYML(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gayle.yml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeAWS(vars map[string]string) func(context.Context, []string) (map[string]string, error) {
	return func(context.Context, []string) (map[string]string, error) {
		return vars, nil
	}
}

const basicYML = `
service: my-service
provider:
  name: ssm
config:
  path: /${stage}/config
  defaults:
    DB_NAME: my-database
    DB_HOST: 3200
  production:
    DB_NAME: my-production-database
    CACHE_TTL: "60"
  required:
    DB_TABLE: some table for ${stage}
secret:
  path: /${stage}/secret
  required:
    DB_PASSWORD: secret database password
`

func TestLoadBasic(t *testing.T) {
	path := writeYML(t, basicYML)
	l := Loader{AWSContext: fakeAWS(map[string]string{"accountId": "123", "region": "us-east-1"})}
	s, err := l.Load(context.Background(), path, nil, "dev")
	if err != nil {
		t.Fatal(err)
	}

	if s.Service != "my-service" || s.Provider.Name != "ssm" {
		t.Errorf("service/provider mismatch: %+v", s)
	}
	if s.Config.Path != "/dev/config" {
		t.Errorf("config path not interpolated: %q", s.Config.Path)
	}
	// YAML integer 3200 must land as the string "3200" (JS coercion parity).
	if s.Config.Defaults["DB_HOST"] != "3200" {
		t.Errorf("DB_HOST = %q, want \"3200\"", s.Config.Defaults["DB_HOST"])
	}
	if s.Config.Required["DB_TABLE"] != "some table for dev" {
		t.Errorf("required description not interpolated: %q", s.Config.Required["DB_TABLE"])
	}
	// Stage overrides: absent for dev, present for production, and never part
	// of the declared parameter list.
	if s.Config.StageOverrides != nil {
		t.Errorf("dev should have no overrides: %v", s.Config.StageOverrides)
	}
	wantConfig := []string{"/dev/config/DB_HOST", "/dev/config/DB_NAME", "/dev/config/DB_TABLE"}
	if !reflect.DeepEqual(s.ConfigParameters, wantConfig) {
		t.Errorf("ConfigParameters = %v, want %v", s.ConfigParameters, wantConfig)
	}
	wantSecret := []string{"/dev/secret/DB_PASSWORD"}
	if !reflect.DeepEqual(s.SecretParameters, wantSecret) {
		t.Errorf("SecretParameters = %v, want %v", s.SecretParameters, wantSecret)
	}
}

func TestLoadStageOverrides(t *testing.T) {
	path := writeYML(t, basicYML)
	l := Loader{AWSContext: fakeAWS(nil)}
	s, err := l.Load(context.Background(), path, nil, "production")
	if err != nil {
		t.Fatal(err)
	}
	if s.Config.StageOverrides["DB_NAME"] != "my-production-database" {
		t.Errorf("production overrides missing: %v", s.Config.StageOverrides)
	}
	// Override-only keys (CACHE_TTL) never join ConfigParameters (Node parity:
	// they are not listed, fetched, or cleaned).
	for _, p := range s.ConfigParameters {
		if strings.HasSuffix(p, "/CACHE_TTL") {
			t.Errorf("override-only key leaked into parameters: %v", s.ConfigParameters)
		}
	}
	if len(s.ConfigParameters) != 3 {
		t.Errorf("ConfigParameters = %v, want the 3 declared keys", s.ConfigParameters)
	}
}

func TestLoadKeyVaultSkipsAWS(t *testing.T) {
	path := writeYML(t, `
provider:
  name: key-vault
  vault: my-vault-${stage}
config:
  path: graph
  defaults:
    A: "1"
`)
	l := Loader{AWSContext: func(context.Context, []string) (map[string]string, error) {
		t.Fatal("AWS context must not be fetched for key-vault")
		return nil, nil
	}}
	s, err := l.Load(context.Background(), path, nil, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if s.Provider.Vault != "my-vault-dev" {
		t.Errorf("vault not interpolated: %q", s.Provider.Vault)
	}
}

func TestLoadCLIVarsAndAWSPrecedence(t *testing.T) {
	path := writeYML(t, `
provider:
  name: ssm
config:
  path: /x
  defaults:
    FROM_CLI: ${foo}
    FROM_CF: ${UserPoolId}
`)
	l := Loader{AWSContext: fakeAWS(map[string]string{"UserPoolId": "pool-1", "foo": "aws-wins"})}
	s, err := l.Load(context.Background(), path, map[string]string{"foo": "cli"}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if s.Config.Defaults["FROM_CF"] != "pool-1" {
		t.Errorf("CF output not interpolated: %v", s.Config.Defaults)
	}
	// Node merge order: {...variables, ...providerContext} — AWS context wins.
	if s.Config.Defaults["FROM_CLI"] != "aws-wins" {
		t.Errorf("precedence mismatch: %v", s.Config.Defaults)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := []struct {
		name    string
		yml     string
		stage   string
		wantErr string
	}{
		{
			name:    "invalid provider",
			yml:     "provider:\n  name: s3\n",
			wantErr: "Invalid provider 's3'!! Only ssm and key-vault are supported.",
		},
		{
			name:    "missing provider",
			yml:     "service: x\n",
			wantErr: "Invalid provider!! 'provider.name' must be set",
		},
		{
			name:    "key-vault without vault",
			yml:     "provider:\n  name: key-vault\n",
			wantErr: "Invalid provider!! 'provider.vault' must be passed for 'key-vault' provider.",
		},
		{
			name:    "undefined variable",
			yml:     "provider:\n  name: ssm\nconfig:\n  path: /${nope}\n  defaults:\n    A: '1'\n",
			wantErr: "nope is not defined",
		},
		{
			name:    "non-identifier expression",
			yml:     "provider:\n  name: ssm\nconfig:\n  path: /${stage + 1}\n  defaults:\n    A: '1'\n",
			wantErr: "unsupported expression ${stage + 1}",
		},
		{
			name:    "declared keys without path",
			yml:     "provider:\n  name: ssm\nconfig:\n  defaults:\n    A: '1'\n",
			wantErr: "config.path must be set",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeYML(t, c.yml)
			l := Loader{AWSContext: fakeAWS(nil)}
			stage := c.stage
			if stage == "" {
				stage = "dev"
			}
			_, err := l.Load(context.Background(), path, nil, stage)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %v, want containing %q", err, c.wantErr)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	l := Loader{AWSContext: fakeAWS(nil)}
	missing := filepath.Join(t.TempDir(), "gayle.yml")
	_, err := l.Load(context.Background(), missing, nil, "dev")
	// The message must name the path that was actually tried — the Node CLI
	// blamed the working directory even when --config pointed elsewhere.
	if err == nil || !strings.Contains(err.Error(), "Could not find gayle.yml at "+missing) {
		t.Errorf("missing-file error mismatch: %v", err)
	}
}

func TestLoadMalformedYAMLIsHonest(t *testing.T) {
	// The Node CLI reported any parse failure as "could not find gayle.yml";
	// the port reports the real problem.
	path := writeYML(t, "provider: [unclosed")
	l := Loader{AWSContext: fakeAWS(nil)}
	_, err := l.Load(context.Background(), path, nil, "dev")
	if err == nil || strings.Contains(err.Error(), "Could not find") {
		t.Errorf("parse error must not masquerade as missing file: %v", err)
	}
}

func TestStringify(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"s", "s"},
		{3200, "3200"},
		{3.5, "3.5"},
		{true, "true"},
		{false, "false"},
		{nil, ""},
	}
	for _, c := range cases {
		got, err := stringify(c.in)
		if err != nil || got != c.want {
			t.Errorf("stringify(%v) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestInterpolateString(t *testing.T) {
	vars := map[string]string{"stage": "dev", "UserPoolId": "p1"}
	got, err := interpolateString("arn-${stage}-${UserPoolId}", vars)
	if err != nil || got != "arn-dev-p1" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := interpolateString("${missing}", vars); err == nil || err.Error() != "missing is not defined" {
		t.Errorf("undefined var error mismatch: %v", err)
	}
	if got, err := interpolateString("no placeholders", vars); err != nil || got != "no placeholders" {
		t.Errorf("passthrough mismatch: %q, %v", got, err)
	}
}
