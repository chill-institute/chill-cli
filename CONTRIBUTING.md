# Contributing

```bash
mise install
mise run hooks   # pre-commit formats staged Go; pre-push runs verify
mise run verify
mise run smoke
```

`chilly` is a human CLI and an agent contract: keep terminal output, JSON,
exit codes, and `--dry-run` previews working together. For command changes,
check `--help`, `schema`, and `--describe` output; update tests and
[the bundled skill](./skills/chilly-cli/SKILL.md); and note compatibility risk
in the pull request.

## Hosted Integration

Opt-in live tests need an API URL and user token, from the environment or an
ignored `.env.local`:

```bash
CHILLY_TEST_API_URL=https://api.chill.institute CHILLY_TEST_TOKEN=... mise run test:integration
```

`main` releases from Conventional Commits ([delivery](./docs/architecture.md#delivery)).
