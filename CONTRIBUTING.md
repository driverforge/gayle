# Contributing

Thanks for helping improve gayle! Contributions are welcome under the project's [MIT license](LICENSE).

## How this repository works

gayle is developed directly in this repository — the normal GitHub workflow applies: open issues and pull requests here, and accepted PRs are squash-merged onto `main`.

## Contributor License Agreement

Contributions require agreeing to our Contributor License Agreement: you keep the copyright to your contribution and grant Driverforge — and its successors and assigns — a licence to use and relicense it. A CLA check runs on your pull request.

**AI assistance is welcome.** However your contribution was produced, you're responsible for it: it must be yours to give, and you must have the right to contribute it under this licence (for example, it must not carry in incompatibly-licensed content). We don't require AI use to be disclosed or attributed — your responsibility for the contribution is the same either way.

## What lives here

gayle is a Go CLI that deploys configuration and secrets to AWS SSM Parameter Store and Azure Key Vault from a `gayle.yml`. The Cobra command tree lives under [`internal/cli`](internal/cli/), the settings pipeline under [`internal/settings`](internal/settings/), and the provider stores under [`internal/paramstore`](internal/paramstore/). Read [`docs/architecture.md`](docs/architecture.md) before making behavioral changes — it maps the packages and records the **deliberately preserved quirks** you should not "fix".

Two contracts to know about up front:

- **The CLI surface is pinned** (commands, flags, defaults, exit-code semantics, the fetch-JSON-on-stdout rule) by `internal/cli/surface_test.go` — CI pipelines invoke gayle unchanged, so surface changes need a deliberate decision and a CHANGELOG entry. Message *wording* is not pinned and may be improved.
- **Exit codes are honest**: 0 strictly means everything verifiably succeeded, 1 is an expected failure, 2 is a crash. Never swallow a provider error; batch operations attempt every key and report each failure.

## Building locally

You need Go (the version in [`go.mod`](go.mod); the `toolchain` directive fetches the right patch release automatically):

```sh
make build      # build the gayle binary
make check      # gofmt check + go vet + tests — CI runs this, run it before pushing
```

Tests need no cloud credentials — the provider stores are tested through in-memory fakes.

## Commit, issue & PR conventions

### Commits

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <subject> (#<issue-or-pr>)
```

e.g. `fix: report per-key errors from Key Vault deletes (#42)`.

- **Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `ci`, `chore`.
- Use the **imperative mood** — "add", not "added".
- Reference the related GitHub issue/PR number in parentheses; it can be left out until the PR is opened.
- Keep commits small and logical, with messages that say what changed and why.

### Issues

For anything beyond a trivial fix, please open an issue first describing the problem or the improvement and who it helps — it's the best place to agree on scope before writing code. For bugs, include your gayle version (`gayle -V`), provider, and a **redacted** config and command output.

### Pull requests

- Title the PR in the same format as a commit (e.g. `fix: mask secrets in clean-up dry-run output`).
- Keep it small and focused; for larger changes, open an issue first.
- `make check` must pass; new behavior needs tests (see the existing table-driven patterns).
- The required checks are the CI build and the CLA.
