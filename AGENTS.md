# CLI

`chilly` is the `chill.institute` terminal client for humans, scripts, and
agents; `pkg/` is the Go surface `chill-mcp` imports. The API owns behavior.

## Work

```bash
mise install
mise run hooks   # pre-push runs mise run verify
mise run verify
```

## Rules

- Preserve stable JSON before terminal polish. Results go to `stdout`, prompts
  and diagnostics to `stderr`; piped output defaults to compact JSON.
- Validate IDs, URLs, procedure names, and file input locally; preview
  mutations with `--dry-run`.
- Keep indexer health tri-state: `healthy`, `degraded`, or `down`.
- `pkg/` must not import `internal/`; its changes follow semantic versioning.
- Update [the bundled skill](./skills/chilly-cli/SKILL.md) with command, auth,
  default, or output changes.
- `releaseRules` in [`.releaserc.json`](./.releaserc.json) match every
  chill.institute package repo; [`smoke.mjs`](./.github/release/smoke.mjs)
  fails on drift.

## Proof map

| Change | Check | CI |
| --- | --- | --- |
| Go, docs, skill, workflows | `mise run verify`: format, lint, 85% coverage, govulncheck, actionlint, zizmor | [Verify](./.github/workflows/verify.yml), [Main](./.github/workflows/main.yml) |
| Command surface or output | `mise run smoke`, then `go run ./cmd/chilly <command> --help` and `--output json` | none |
| Schema metadata | `mise run contracts:check` against `../chill-contracts` | Verify, Main, pinned tag |
| Release rules | `npm ci --prefix .github/release --ignore-scripts && node .github/release/smoke.mjs` | Verify |
| GoReleaser config | `goreleaser release --snapshot --clean` twice; `dist/checksums.txt` must match | none |
| Hosted API | `mise run test:integration` ([setup](./CONTRIBUTING.md#hosted-integration)) | none |
| Release | push to `main`; recover with a Main dispatch with `tag` (`dry_run` defaults on) | Main `release`, `publish` |

Gaps:

- Smoke uses `go run` and runs only locally; no lane runs a packaged binary.
- Release shell steps run only in a real release or a `dry_run` dispatch.
- Hosted integration has no CI runner or test account.
- No check covers Markdown links.
