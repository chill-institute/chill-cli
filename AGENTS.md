# CLI

`chilly` is a human CLI and agent-facing SDK for `chill.institute`.

## Work

```bash
mise install
mise run hooks
```

The pre-push hook runs `mise run verify`.

## Proof map

| Change | Check | Runs | Leaves |
| --- | --- | --- | --- |
| Docs, skill, config | `mise run verify` | local, pre-push, [PR](./.github/workflows/verify.yml), [main](./.github/workflows/main.yml) `verify` | exit status |
| Go logic, flags, validation | `mise run verify` (format, tidy, lint, tests at 85% coverage, govulncheck) | local, pre-push, [PR](./.github/workflows/verify.yml), [main](./.github/workflows/main.yml) `verify` | exit status, `coverage.out` |
| Command surface or output contract | `mise run smoke`, then `go run ./cmd/chilly <command> --help` and `--output json` | local | exit status, stdout |
| Schema metadata | `mise run contracts:check` against `../chill-contracts` | local, [PR](./.github/workflows/verify.yml), [main](./.github/workflows/main.yml) `verify` at a pinned tag | exit status |
| Workflows | `mise run actions` (actionlint, zizmor; inside verify) | local, CI with verify | exit status |
| Hosted API behavior | `mise run test:integration`; scope in [Hosted Integration](./CONTRIBUTING.md#hosted-integration) | local only, `CHILLY_TEST_API_URL` and `CHILLY_TEST_TOKEN` | exit status |
| Release and packaging | push to `main`; recover with the [Release](./.github/workflows/release.yml) workflow | [main](./.github/workflows/main.yml) `release`, `publish` | tag, immutable GitHub release with 7 assets and provenance, Homebrew formula, npm packages |

Each commit type listed in [`.releaserc.json`](./.releaserc.json),
including `docs`, publishes a release from `main`.

Gaps:

- No lane runs the packaged binary from GoReleaser, npm, or Homebrew; smoke
  uses `go run`. Owner: chill-institute/chill-cli.
- `mise run smoke` does not run in CI. Owner: chill-institute/chill-cli.
- Hosted integration has no CI runner or declared test account. Owner:
  operator.
- No Markdown or link check covers docs and the bundled skill. Owner:
  chill-institute/chill-cli.

## Contracts

- Preserve stable JSON before refining terminal presentation.
- Piped results default to compact JSON unless `--output` is explicit.
- Keep command results on `stdout`; prompts and diagnostics belong on `stderr`.
- Validate opaque IDs, URLs, procedure names, and filesystem input locally.
- Preview mutations with `--dry-run` where the command supports it.
- Keep indexer health tri-state: `healthy`, `degraded`, or `down`.
- Update [the bundled skill](./skills/chilly-cli/SKILL.md) with command, auth,
  default, or output-contract changes.

## Ownership

- Cobra commands and orchestration: `internal/cli/`
- Public procedure names and input validation: `pkg/chill/`
- Local profiles and credentials: `pkg/config/`
- API transport and error mapping: `pkg/rpc/`
- Release lookup and binary replacement: `internal/update/`
- Shared tasks and hook behavior: `mise.toml`

[Architecture](./docs/ARCHITECTURE.md) · [Contributing](./CONTRIBUTING.md) ·
[Security](./SECURITY.md)
