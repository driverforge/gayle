# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Fixed
- **`clean-up` (and `run --removing`) no longer deletes stage-override-only
  config keys.** Keys declared only under `config.<stage>` are written by
  `run` (defaults + overrides merged) but were excluded from the declared
  set the pruner diffs against, so every `run --removing` deleted the
  parameters it had just written — in production this purged all stage-only
  config for two deployed services before being caught (driverforge
  DF-659). The pruner now counts `config.<stage>` keys as declared. The
  v5-parity exclusion of override-only keys from `list`/`fetch` scope is
  unchanged.
- `clean-up` no longer lists a shared path twice when `config.path` and
  `secret.path` are identical (the norm for Key Vault declarations). The
  duplicate listing queued every orphan for a second delete, which 404'd
  and failed the run.
- Key Vault: a 404 on delete is treated as already-pruned instead of a
  per-key failure. SSM behavior (missing names reported via
  `InvalidParameters`) is unchanged.

## [6.0.0](2026-07-15)

Gayle is now a Go binary. The Node.js implementation and npm distribution are
retired; the CLI surface (commands, flags, defaults, output wording, the
fetch-JSON-on-stdout contract) is unchanged. Install via
`brew install driverforge/tap/gayle`, Scoop, or the archives on
releases.driverforge.com — see the README. The `@driverforge/gayle` npm
package is deprecated and frozen at v5.

### Changed
- **Exit codes are honest end-to-end.** Exit 0 strictly means every operation
  verifiably succeeded; expected failures exit 1; crashes exit 2.
  - usage errors (unknown command/flag, missing `--stage`, `fetch` without
    `-k`) exit 1 — v5 printed help and exited 0
  - partial provider write/delete failures attempt every key, report each
    failed key, and exit 1
  - Key Vault read errors other than 404 (auth/network/throttle) fail the run
    instead of silently reading as empty values
  - a CloudFormation DescribeStacks failure is a hard error instead of a
    warning with empty outputs
  - `fetch` errors on keys not declared in the configuration instead of
    silently omitting them
  - a malformed gayle.yml reports the real parse error instead of
    "Could not find gayle.yml"
  - SSM deletes now surface `InvalidParameters` from the delete response
- `-r/--removing`, `-d/--dry-run`, `-C/--config-only` are real boolean flags
  (`-r` or `--removing=false`). v5's optional-value quirk meant `-r false`
  still deleted; that form is now a usage error.
- `${...}` interpolation supports bare variable names only (v5's lodash
  template technically evaluated JavaScript; no known config relied on it)
- interactive prompts require a terminal: `run -i` under CI or a pipe fails
  fast instead of hanging
- a `secret.keyId` in gayle.yml now logs a warning: it was documented in v5
  but always ignored — SSM secrets are encrypted with `alias/aws/ssm`

### Added
- `clean-up` (and `run --removing`) refuses to prune when the configuration declares no config or secret keys — an empty or misparsed gayle.yml can no longer delete every remote parameter under the app's paths (DF-644)
- each pruned parameter is logged by name in non-dry-run mode

### Fixed
- Key Vault deletion failures are no longer silently swallowed: a failed delete rejects the run (the secret is still live); a failed purge only warns (soft-deleted secrets no longer appear in listings)
- `import` with an empty `configs` or `secrets` section no longer crashes
- `generate`'s file write is checked (v5 swallowed write errors)
- `export --target` is validated (json|env)
- misleading messages corrected: a missing config file names the path that was
  actually tried (v5 always blamed the working directory, even with
  `--config`); a missing `provider.name` no longer prints `'undefined'`;
  "Please specify ssmPath…" (wrong key name, wrong for Key Vault) and the
  bare "Missing path!" now name the actual `config.path`/`secret.path` keys

## [v5.3.1](2022-05-16)
### Fixed
- using `-m` when there's no missing secrets doesn't crash anymore

## [v5.3.0](2021-09-06)

### Added

- `export` command now supports `-C, --config-only` option to only export configurations

## [v5.2.1](2021-08-28)

### Added

- `run` command now supports `-r, --removing` option to remove orphan configurations or secrets

## [v5.2.0](2021-08-27)

### Added

- `clean-up` command

## [v5.0.0](2021-06-18)

### Removed

- Dropped support for `DDB` provider

## [v4.0.3](2021-04-01)

### Removed

- Dropped support for `node 8` and `10`

## [v3.0.0](2019-11-01)

### Added

- Prevent SSM from writing values unless it has changed

### Removed

- Dropped support for `node 6`

## [v2.7.0](2019-04-23)

### Added

- `-m, --missing` To prompt only missing values in interactive mode

## [v2.6.0](2019-04-02)

### Added

- `fetch` command. You can now fetch keys from CLI

## [v2.5.0](2019-04-01)

### Added

- `accountId` Can now be used to fetch the aws accountId the configuration is deployed to

## [v2.4.0](2019-02-12)

### Fixed

- `cfOutputs` allow CloudFormation stack outputs to be pushed to the parameter store

## [v2.3.1](2019-02-04)

### Added

- `import` & `export` command to import/export key values from specified provider

## [v2.3.0](2019-02-01)

### Changed

- `run` command now runs both `init` and `configure`.

## [v2.2.1](2019-01-31)

### Fixed

- Terminate process with exit code 1 on errors

## [v2.2.0](2019-01-29)

### Changed

- Retains DynamoDB by default to prevent accidental deletion

## [v2.1.0](2019-01-29)

### Changed

- Mask secrets when parameters are listed

## [v2.0.0](2019-01-24)

### Added

- Add support to DynamoDB provider

## [v1.0.0](-----)

### Added

- Initial Release
