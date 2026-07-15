*Originally forked from [@acloudguru/oprah](https://github.com/ACloudGuru/oprah)*

# Gayle

CLI to push configuration and encrypted secrets to AWS SSM Parameter Store and Azure Key Vault, driven by a `gayle.yml`.

Gayle is a single static Go binary. Version 6 replaced the Node.js implementation (`@driverforge/gayle` on npm) — same commands, same flags, same output; only the installation changed. See [Changes from v5](#changes-from-v5-node) if you're migrating.

## Installation

**macOS / Linux (Homebrew):**

```
brew install driverforge/tap/gayle
```

**Windows (Scoop):**

```
scoop bucket add driverforge https://github.com/driverforge/scoop-bucket
scoop install gayle
```

**Direct download (CI pipelines):**

Archives for linux/darwin/windows on amd64/arm64, with checksums, are published to
`https://releases.driverforge.com/driverforge-releases/gayle/<tag>/` — the
`latest/manifest.json` alongside them lists the current version and artifact URLs:

```bash
curl -fsSL https://releases.driverforge.com/driverforge-releases/gayle/latest/manifest.json
```

**From source:**

```
go install github.com/driverforge/gayle/cmd/gayle@latest
```

## Usage

1. At the root of your application add a configuration file called `gayle.yml` (`gayle generate` writes an example).
2. Use the `gayle` CLI to push your keys to your provider.

```
$ gayle run --stage <stage> --interactive
```

### Provider Examples

#### AWS SSM (Systems Manager Parameter Store)

```yaml
service: my-service
provider:
  name: ssm

config:
  path: /${stage}/config
  defaults:
    DB_NAME: my-database
    DB_HOST: 3200
  required:
    DB_TABLE: "some database table name for ${stage}"

secret:
  path: /${stage}/secret
  required:
    DB_PASSWORD: "secret database password"
```

#### Azure Key Vault

The `vault` property specifies the Azure Key Vault name. Authentication uses Azure's `DefaultAzureCredential`. Key Vault paths must not contain slashes (secret names are built as `<path>--<KEY>` and Azure only allows alphanumerics and hyphens).

```yaml
service: my-service
provider:
  name: key-vault
  vault: my-vault-${stage}                    # Azure Key Vault name (used to build https://<vault>.vault.azure.net)

config:
  path: graph
  defaults:
    DB_NAME: my-database
    DB_HOST: 3200
  required:
    DB_TABLE: "some database table name for ${stage}"

secret:
  path: graph
  required:
    DB_PASSWORD: "secret database password"
```

### Config File

Following is the configuration file with all possible options:

```yaml
service: my-service
provider:
  name: ssm                                   # Supports ssm and key-vault.
  # vault: my-vault-${stage}                  # Required for key-vault provider.

stacks:                                       # Outputs from cloudformation stacks that needs to be interpolated.
  - some-cloudformation-stack

config:
  path: /${stage}/config                      # Base path for params to be added to
  defaults:                                   # Default parameters. Can be overwritten in different environments.
    DB_NAME: my-database
    DB_HOST: 3200
  production:                                 # If keys are deployed to production stage, its value will be overwritten by following
    DB_NAME: my-production-database
  required:                                   # Keys mentioned below will be prompted to be entered.
    DB_TABLE: "some database table name for ${stage}"

secret:
  path: /${stage}/secret                      # Base path for params to be added to
  required:
    DB_PASSWORD: "secret database password"   # Parameter to encrypt and add to. Will be encrypted using KMS.
                                              # Above key will be added to /${stage}/secret/DB_PASSWORD
                                              # Value in quote will be displayed as explanation in prompt during interactive run.
```

Interpolation: `${name}` references are replaced from the stage (`${stage}`), any `-v/--variables` JSON, and — for the ssm provider — `${accountId}`, `${region}`, and every CloudFormation output of the stacks listed under `stacks:`. A reference with no value is an error. Only bare variable names are supported inside `${...}`.

> **Note:** SSM secrets are always encrypted with the account-default `alias/aws/ssm` KMS key. A `secret.keyId` in the yml is not supported (v5 documented it but also ignored it); gayle warns if it is set.

### CLI

Following is all options available in the `gayle` CLI.

```
Usage:
  gayle [command]

Available Commands:
  clean-up    Cleaning up orphan configs or secrets
  export      Export of all of the configuration from the provider to a text json file
  fetch       Fetch config or secret
  generate    Generate an example configuration file.
  import      Import all of the configuration from the json from to a provider
  init        Initialize gayle. Only required to run once.
  list        List all remote configurations and secrets.
  run         Verify or populate all remote configurations and secrets.

Flags:
  -c, --config string   Path to gayle configuration (default "gayle.yml")
  -h, --help            help for gayle
  -s, --stage string    Specify stage to run on. (required)
  -V, --version         output the version number
```

### Push configuration

```
Usage:
  gayle run [flags]

Flags:
  -i, --interactive        Run on interactive mode
  -m, --missing            Only prompt missing values in interactive mode
  -r, --removing           Removing orphan configs or secrets
  -v, --variables string   Variables used for config interpolation.
```

In non-interactive mode (the CI default), `run` writes the declared defaults and stage overrides, and **verifies** every `required` key already holds a remote value — it never invents values. Missing required keys are reported per key and the run exits 1.

### List pushed configurations

```
Usage:
  gayle list [flags]
```

Secrets are printed masked (all but the last four characters).

### Fetch individual configuration

```
Usage:
  gayle fetch [flags]

Flags:
  -k, --keys string   Comma separated configs to fetch (example: "SOME_CONFIG,ANOTHER_CONFIG")
```

The JSON result is the only thing gayle ever writes to stdout — all diagnostics go to stderr — so it pipes cleanly:

```bash
PARAMS=$(gayle fetch -k "CALLBACK_URL,LOGOUT_URL" -s $STAGE)

CALLBACK_URL=$(echo $PARAMS | jq -er ".CALLBACK_URL")
LOGOUT_URL=$(echo $PARAMS | jq -er ".LOGOUT_URL")

# do something with the values
```

### Import

```
Usage:
  gayle import [flags]

Flags:
  -p, --path string   The location of the secrets and configuration file (default: "/tmp/gayle-exports.json")
```

### Export

```
Usage:
  gayle export [flags]

Flags:
  -C, --config-only     Only export configs
  -p, --path string     The location for the output secrets & configuration file (default: "/tmp/gayle-exports.json" or ".env_gayle")
  -t, --target string   The output target, available options are json|env (default:json)
```

> The `env` target writes values raw and unescaped inside double quotes — a value containing `"` or a newline breaks the file. This matches v5; treat the output as trusted-input-only.

### Clean up

```
Usage:
  gayle clean-up [flags]

Flags:
  -d, --dry-run   Execute a dry run
```

Deletes every remote parameter under the configured paths that is no longer declared in `gayle.yml`. Refuses to run when the configuration declares no keys at all (an empty declaration would prune everything). Requires both `config.path` and `secret.path`.

## Changes from v5 (Node)

The v6 rewrite keeps the CLI surface and output stable, with one deliberate theme: **exit codes are honest**. Exit 0 strictly means everything verifiably succeeded; expected failures exit 1; crashes exit 2.

- Usage errors (unknown command/flag, missing `--stage`, missing `-k`) now exit 1 — v5 printed help and exited 0.
- Partial provider failures report every failed key and exit 1; the run attempts all keys first.
- Key Vault read errors other than 404 (auth, network, throttling) fail the run — v5 silently read them as empty values.
- A CloudFormation `DescribeStacks` failure is a hard error — v5 swallowed it into a warning and empty outputs.
- `fetch` errors on keys not declared in the configuration — v5 silently omitted them from the JSON.
- `import` tolerates an empty `configs`/`secrets` section — v5 crashed.
- `export --target` is validated (`json`/`env`); `generate`'s file write is checked.
- `-r/--removing`, `-d/--dry-run`, and `-C/--config-only` are real boolean flags: use `-r` or `--removing=false`. v5's optional-value quirk meant `-r false` still deleted; that form is now a usage error.
- A malformed `gayle.yml` reports the actual parse error — v5 claimed the file didn't exist.

Installation is via Homebrew/Scoop/direct download (above). The `@driverforge/gayle` npm package is deprecated and frozen at v5.

### License

Feel free to use the code, it's released using the MIT license.
