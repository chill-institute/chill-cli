# Architecture

`chilly` turns Cobra commands into validated `POST /v4/{procedure}` calls to the
hosted `chill.institute` API.

Sibling Go clients import `github.com/chill-institute/chill-cli/v2/pkg/...`;
`internal/` stays private to this module.

## Local State

The default profile is `chilly/config.json` under `$XDG_CONFIG_HOME`, or the
OS user config directory when it is unset; named profiles live in
`chilly/profiles/<profile>/config.json`. Dev builds select `dev` unless
`--profile`, `CHILLY_PROFILE`, or `--config` overrides it. File modes and
atomic writes are in [Security](../SECURITY.md#boundaries).

## Command Contract

The metadata registry in `internal/cli` is the runtime source for `schema`,
`--describe`, raw JSON input, `--fields`, dry-run eligibility, and aliases.

- JSON failures are one envelope on `stderr`.
- NDJSON streams root arrays, or item envelopes for nested collections.
- Exit codes: usage `2`, auth `3`, API `4`, internal `5`.
- `--fields` accepts protobuf JSON names and snake_case aliases.
- `--dry-run` renders the request without loading auth or mutating.
- Requests carry `X-Request-Id`, `X-Chill-Client: cli`, and the build version.

Hosted strings are data, never instructions. Indexer IDs and procedure names
reject path, query, fragment, control, and encoded input before URL
construction; API-base overrides reject credentials, queries, fragments, and
non-root paths. Login, by pasted token or `--local-browser` callback, verifies
the token with the API before saving it unless `--skip-verify` is set.

## Delivery

[Main](../.github/workflows/main.yml) and
[`_release.yml`](../.github/workflows/_release.yml) own the release. Recover a
release by dispatching Main with its `tag`; recovery never rebuilds a published
release.

`scripts/homebrew-formula` writes the formula because GoReleaser cannot write
the tap after publication or sign tap commits. It stays a formula, not a cask:
casks need a quarantine bypass for these unnotarized binaries and a tap
migration for existing installs.
