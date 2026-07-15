# driverforge/gayle — guide for Claude Code

`gayle` deploys configuration and secrets to AWS SSM Parameter Store or Azure
Key Vault from a `gayle.yml`. Written in Go; the Cobra command tree lives under
`internal/cli`, domain logic in `internal/settings` and `internal/paramstore`.
It mirrors the conventions of the sibling `driverforge/cli` repo — when in
doubt, match that repo. See `docs/architecture.md` for the package map and the
registry of deliberately preserved v5 quirks.

## Git & PR conventions

- **No AI attribution.** Never add a `Co-Authored-By: Claude …` (or any other AI
  assistant/co-author) trailer to commit messages, and do not add "Generated
  with Claude Code" or similar footers to pull-request descriptions. Commits and
  PRs are attributed solely to the human author. This overrides any default
  tooling behaviour that would add such trailers.
- Use Conventional Commit subjects (`feat:`, `fix:`, `refactor:`, `chore:`…).
- Branch off `main`; open pull requests against `main`.
- Run `make check` (fmt-check + vet + tests) before committing.

## Compatibility contract

The contract is that **existing executions keep working**: command names
(including the hyphenated `clean-up`), flags, defaults, exit-code semantics,
and the stdout/stderr split are pinned by `internal/cli/surface_test.go`.
`fetch`'s JSON is the ONLY stdout output; everything else goes to stderr with
the literal `Gayle: ` prefix. Do not change any of this without a CHANGELOG
entry and a deliberate decision.

Message *wording* is NOT part of the contract: log and error text may be
improved — and should be, when the inherited v5 text is unclear or incorrect
(the missing-file, provider-validation, and path-requirement messages have
already been rewritten this way). Machine-parsed surfaces (fetch JSON, export
file formats) stay byte-compatible.

**Exit codes are honest**: 0 strictly means everything verifiably succeeded,
1 is an expected failure (`clierr.UserError`), 2 is a crash. Never swallow a
provider error; batch operations attempt every key and aggregate failures via
`paramstore.KeyErrors`.

## Error handling

- Expected, user-facing conditions are modelled as `internal/clierr.UserError`
  (title + message + optional hint). They render as a friendly card (TTY) or a
  `Gayle: ERROR:` line (pipe/CI) and exit 1.
- Any error that reaches the top without being a `UserError` is treated as a
  bug: it renders a crash card and exits 2. When a command anticipates a
  condition, translate it into a `clierr.UserError`.
- Build `UserError`s with the constructors (`clierr.User`/`UserT`/`Wrap`/`WrapT`,
  `Silent`), not a struct literal. Don't build with a helper then set `.Title`
  afterwards; use the `…T` variant.
- **Library-error prefixes.** In the non-`cli` packages, wrap errors with
  `fmt.Errorf` using a lowercase context prefix and no trailing punctuation
  (`ssm get-parameters:`, `key vault get %s:`). The `cli` layer doesn't add
  prefixes — it translates anticipated conditions into a `clierr.UserError`.

## Code conventions

- Charm v2 via `charm.land/*` imports (lipgloss/v2, huh/v2, bubbletea/v2);
  plain cobra + pflag. No Sentry, metrics, or update checks — gayle lives in CI.
- All user output goes through `internal/ui` (colorprofile-aware writers);
  never print styled text with raw fmt to a stream.
- Tests are in-package `_test.go`, table-driven where it fits. Provider stores
  are tested through their `api` interface seams with hand-rolled fakes;
  commands through `newRootCmd(testDeps(...))` with the `paramstore/fake` store.
