# gayle architecture

gayle is a Go port (v6) of the original Node CLI. The port is behavior-first:
the CLI surface and output are pinned to v5, with a set of deliberate fixes
under one theme — **honest exit codes**. This doc maps the packages and records
the quirks that were preserved on purpose, so nobody "fixes" them by accident.

## Package map

```
cmd/gayle            main() → cli.Execute() → exit code
internal/cli         Cobra tree; thin adapters over the domain packages.
                     deps.go is the seam commands share: memoized settings
                     load + provider store construction (tests inject both).
                     surface_test.go pins the v5 CLI surface.
internal/settings    gayle.yml pipeline: read → validate provider (raw tree) →
                     gather AWS context (STS accountId, region, CF stack
                     outputs; skipped entirely for key-vault) → interpolate
                     every scalar (${name} only, JS-style scalar coercion:
                     3200 → "3200") → derive ConfigParameters/SecretParameters
                     ("<path>/<KEY>", sorted).
internal/paramstore  Provider-neutral Store interface + KeyErrors (per-key
                     failure aggregation) + fake/ (in-memory store for tests).
  ssm/               aws-sdk-go-v2. Chunk-of-10 batched reads with a per-run
                     cache; NextToken pagination; prefetch-diff writes;
                     SecureString hard-wired to alias/aws/ssm.
  keyvault/          azsecrets + DefaultAzureCredential. Name mangling
                     (service--KEY, _↔-); type tag (config|secret) maps onto
                     String/SecureString; whole-vault list + prefix filter;
                     delete = DeleteSecret → poll GetDeletedSecret → purge.
internal/clierr      UserError (expected failure → exit 1). Everything else
                     that reaches the top is a crash → exit 2.
internal/ui          All output. Log lines: stderr, "Gayle: " prefix,
                     chalk-equivalent colors via lipgloss v2 + colorprofile
                     (auto-plain on pipes/NO_COLOR). Error cards on TTYs,
                     "Gayle: ERROR:" lines otherwise. huh prompt for -i.
```

## Store contract (honesty rules)

- **Reads**: only a definitively missing parameter (SSM `InvalidParameters`,
  Key Vault 404) maps to `""` — that emptiness drives the missing-required
  flow. Any other error (auth, network, throttle) fails the read.
- **Writes/deletes**: sequential in sorted name order (deterministic logs),
  attempt every key, aggregate failures into `paramstore.KeyErrors`, return
  nil only when everything verifiably succeeded. Writes skip values that
  already match the remote store (no version churn) via a prefetched diff —
  this replaces the Node DataLoader.
- **Key Vault delete asymmetry (DF-644)**: a failed or unconfirmed delete is a
  hard per-key error (the secret is still live); a failed purge is a warning
  only (purge protection/RBAC forbid it legitimately, and the soft-deleted
  secret no longer appears in listings, which is all pruning needs).

## Quirk registry — preserved v5 behavior (do not "fix")

| Quirk | Where | Why it stays |
|---|---|---|
| `defaults` override freshly prompted values (merge: prompt < defaults < stage overrides) | cli/configure.go | Pipelines rely on defaults winning; changing it changes deployed values. |
| Stage-override blocks affect only config population; override-only keys are NOT in ConfigParameters (not listed/fetched/cleaned) | settings/derive.go | Changing it changes list/fetch/clean-up scope. |
| `secret.keyId` ignored; SSM secrets always `alias/aws/ssm` (warning when set) | paramstore/ssm/put.go, cli/deps.go | Honoring it would silently re-encrypt existing parameters. |
| env export values unescaped inside `"…"` | cli/export.go | Existing consumers parse the v5 format. |
| Export/import defaults `/tmp/gayle-exports.json` / `.env_gayle` | cli/export.go | Pipelines hardcode the paths. |
| Mask reveals last 4 chars (`/\S(?=\S{4})/g` port, whitespace lookahead quirks included) | ui/mask.go | Output parity; pinned against Node in tests. |
| `Gayle: ` stderr prefix and the v5 log-line shapes (`Updated config: Name: …`, `Cleanup --> …`) | ui/log.go, throughout | Pipelines grep stderr. Message *wording* is not contractual and may be improved when the v5 text was unclear or wrong (missing-file, provider validation, and path-requirement messages already were). |
| fetch JSON is the only stdout output | cli/fetch.go | `gayle fetch … \| jq` is the documented scripting interface. |
| `init` is a settings-validating no-op | cli/init.go | Pipelines still call it. |
| clean-up requires BOTH config.path and secret.path (`Missing path!`) | cli/cleanup.go | v5 parity; loosening widens the delete surface. |
| Key Vault name mangling is lossy (`-`→`_` on the way back) and multi-segment SSM-style paths produce Azure-invalid names | paramstore/keyvault/keyutils.go | Existing vault contents were written with this scheme. |

## Deliberate deviations from v5

The full list lives in the CHANGELOG under 6.0.0. Summary: usage errors exit 1,
partial failures report per-key and exit 1, non-404 Key Vault read errors and
CF DescribeStacks failures are hard errors, fetch rejects undeclared keys,
import tolerates empty sections, generate's write is checked, `-r/-d/-C` are
real bool flags, `${…}` accepts bare identifiers only, `-i` without a TTY fails
fast, malformed yml reports the real parse error, crashes exit 2.

## Release

GoReleaser (`.goreleaser.yaml`) on `v*` tags: archives to Azure blob
`driverforge-releases/gayle/<tag>/` (canonical, served at
releases.driverforge.com), Homebrew cask in `driverforge/homebrew-tap`, Scoop
manifest, GitHub Release as mirror, `manifest.json` published to the tag path
and `latest/`. The pipeline is a clone of `driverforge/cli`'s — keep them
aligned.
