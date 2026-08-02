# Contributing

Thanks for contributing to `chill-cli`.

## Setup

Install the toolchain:

```bash
mise install
mise run hooks
```

Build the CLI locally:

```bash
go build ./cmd/chilly
```

Run from source while developing:

```bash
go run ./cmd/chilly version --output json
```

## Validation

Run the full repo checks before opening or updating a pull request:

```bash
mise run verify
```

Format Go files when needed:

```bash
mise run fmt
```

Other shared development commands:

```bash
mise run smoke
mise run fmt:check
mise run lint
mise run coverage
mise run coverage:report
mise run security
```

The repository ships git hooks in `.githooks/`:

- `pre-commit` launches `mise run hooks:pre-commit`, which formats staged Go files with `goimports` and `gofmt`
- `pre-push` launches `mise run hooks:pre-push`, which runs `mise run verify`

The hook files stay as tiny executable launchers because Git requires hook entrypoints to be executable files. The actual workflow logic lives in `mise.toml`

Opt-in live integration checks are available when you want to verify the real hosted API surface:

```bash
CHILLY_TEST_API_URL=https://api.chill.institute \
CHILLY_TEST_TOKEN=... \
mise run test:integration
```

## Development Notes

- `chilly` is the command-line client for `chill.institute`
- Prefer explicit, scriptable CLI behavior.
- Keep human output and machine-readable output both in mind.
- Keep `skills/chilly-cli/` in sync when install flows, auth behavior, schema output, or command surfaces change.

## Pull Requests

- Keep commands and flags stable unless a change is clearly justified.
- Update docs or examples when UX changes.
- Add or update tests when behavior changes.

## Release Flow

Normal CLI change flow:

1. Make the change.
2. Run `mise run verify`
3. Open or update a pull request. GitHub Actions runs `Verify` on pull requests.
4. Merge to `main`, where GitHub Actions runs `Main`, re-verifies the repo, runs `semantic-release`, creates the next `vX.Y.Z` tag from conventional commits, and then runs GoReleaser to publish release archives, create the GitHub release, update the Homebrew tap, and publish npm packages under `@chill-institute/cli`
5. Operators can use the tag-based `Release` workflow to republish GitHub and Homebrew artifacts for an intentionally pushed tag. npm trusted publishing runs through `Main`.

The npm package name is `@chill-institute/cli`; its installed binary remains `chilly`.

Versioning notes:

- Releases are driven by conventional commits.
- `feat:` produces a minor release.
- `fix:`, `perf:`, `refactor:`, `docs:`, `test:`, `build:`, `ci:`, and `chore:` produce a patch release.
- Breaking changes produce a major release.
