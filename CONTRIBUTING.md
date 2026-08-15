# Contributing

`chilly` is both a human CLI and an agent-facing contract. Preserve interactive
output, JSON behavior, exit codes, and safe mutation previews together.

## Start

```bash
mise install
mise run hooks
go run ./cmd/chilly version --output json
mise run verify
```

Useful focused checks:

```bash
mise run smoke
mise run fmt
mise run coverage:report
mise run security
```

Git hooks delegate to `mise.toml`: pre-commit formats staged Go files and
pre-push runs the canonical verification gate.

## Hosted Integration

Live tests are opt-in and require an API URL and user token:

```bash
CHILLY_TEST_API_URL=https://api.chill.institute \
CHILLY_TEST_TOKEN=... \
mise run test:integration
```

## Change the Command Surface

- Check `chilly <command> --help`, `schema`, and `--describe` output.
- Keep result data on `stdout`; prompts and diagnostics belong on `stderr`.
- Update tests, examples, and `skills/chilly-cli/` with user-facing changes.
- Explain compatibility or migration risk in the pull request.

Merges to `main` are released from Conventional Commits. Automation creates the
tag, immutable GitHub release, Homebrew formula, and npm packages. The manual
`Release` workflow recovers an existing release tag.
