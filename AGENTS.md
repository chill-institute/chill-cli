# CLI

`chilly` is a human CLI and agent-facing SDK for `chill.institute`.

## Work

```bash
mise install
mise run hooks
mise run smoke
mise run verify
```

For command changes, also inspect `go run ./cmd/chilly <command> --help`.
Hosted integration tests are opt-in through `mise run test:integration`.

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
- Local profiles and credentials: `internal/config/`
- API transport and error mapping: `internal/rpc/`
- Release lookup and binary replacement: `internal/update/`
- Shared tasks and hook behavior: `mise.toml`

[Architecture](./docs/ARCHITECTURE.md) · [Contributing](./CONTRIBUTING.md) ·
[Security](./SECURITY.md)
