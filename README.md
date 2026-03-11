*Originally forked from [@acloudguru/oprah](https://github.com/ACloudGuru/oprah)*

# Gayle

Node module to push configuration and encrypted secrets to AWS and Azure.

## Installation

```
# Via yarn
$ yarn add @driverforge/gayle

# Via npm
$ npm install @driverforge/gayle
```

## Usage

1. At the root of your application add configuration file called `gayle.yml`.
2. Use `gayle` CLI tool to push your keys to your provider.

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
  keyId: some-arn-of-kms-key-to-use           # If not specified, default key will be used to encrypt variables.
  path: /${stage}/secret
  required:
    DB_PASSWORD: "secret database password"
```

#### Azure Key Vault

The `vault` property specifies the Azure Key Vault name. Authentication uses Azure's `DefaultAzureCredential`.

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
  keyId: some-arn-of-kms-key-to-use           # If not specified, default key will be used to encrypt variables. (SSM only)
  path: /${stage}/secret                      # Base path for params to be added to
  required:
    DB_PASSWORD: "secret database password"   # Parameter to encrypt and add to. Will be encrypted using KMS.
                                              # Above key will be added to /${stage}/secret/DB_PASSWORD
                                              # Value in quote will be displayed as explanation in prompt during interactive run.
```

### CLI

Following is all options available in `gayle` CLI.

```
Usage: gayle [options] [command]

Options:
  -V, --version          output the version number
  -s, --stage [stage]    Specify stage to run on. (required)
  -c, --config [config]  Path to gayle configuration (default: "gayle.yml")
  -i, --interactive      specify values through command line
  -h, --help             display help for command

Commands:
  run [options]          Verify or populate all remote configurations and
                         secrets.
  init                   Initialize gayle. Only required to run once.
  export [options]       Export of all of the configuration from the provider
                         to a text json file
  import [options]       Import all of the configuration from the json from to
                         a provider
  list                   List all remote configurations and secrets.
  fetch [options]        Fetch config or secret
  help [command]         display help for command
```

### Push configuration

```
Usage: gayle run [options]

Verify or populate all remote configurations and secrets.

Options:
  -v, --variables [variables]  Variables used for config interpolation.
  -i, --interactive            Run on interactive mode
  -m, --missing                Only prompt missing values in interactive mode
  -r, --removing               Removing orphan configs or secrets
  -h, --help                   display help for command
```

### List pushed configurations

```
Usage: gayle list [options]

List all remote configurations and secrets.

Options:
  -h, --help  display help for command
```

### Fetch individual configuration

```
Usage: gayle fetch [options]

Fetch config or secret

Options:
  -k, --keys [keys]  Comma seperated configs to fetch (example:
                     "SOME_CONFIG,ANOTHER_CONFIG")
  -h, --help         display help for command
```

Fetch configuration can be used in automation scripts. Example:

```bash
PARAMS=$(./node_modules/.bin/gayle fetch -k "CALLBACK_URL,LOGOUT_URL" -s $STAGE)

CALLBACK_URL=$(echo $PARAMS | jq -er ".CALLBACK_URL")
LOGOUT_URL=$(echo $PARAMS | jq -er ".LOGOUT_URL")

# do something with the values
```

### Import

```
Usage: gayle import [options]

Import all of the configuration from the json from to a provider

Options:
  -p, --path [path]  The location of the secrets and configuration file
                     (default: "/tmp/gayle-exports.json")
  -h, --help         display help for command
```

### Export

```
Usage: gayle export [options]

Export of all of the configuration from the provider to a text json file

Options:
  -p, --path [path]      The location for the output secrets & configuration file
                         (default: "/tmp/gayle-exports.json" or ".env_gayle")
  -t, --target [target]  The output target, available options are json|env
                         (default:json)

  -C, --config-only [configOnly] Only export `config` section

  -h, --help             display help for command
```

### Clean up

```
Usage: gayle clean-up [options]

Clean up orphan configurations and secrets from provider

Options:
  -d, --dry-run [dryRun]  Execute a dry run to display all orphan configurations and secrets

  -h, --help              display help for command
```

### License

Feel free to use the code, it's released using the MIT license.
