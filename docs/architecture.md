# Architecture

`chilly` turns Cobra commands into validated `POST /v4/{procedure}` calls to the
hosted `chill.institute` API.

Sibling Go clients import `github.com/chill-institute/chill-cli/v2/pkg/...`;
`internal/` stays private to this module.

## Local State

Use `chilly settings path` to locate a profile and `chilly settings show` to
inspect its candidate hosts (`api_base_urls`, in try order). Profile resolution
and host order live in
[pkg/config/store.go](../pkg/config/store.go); file modes and atomic writes
are in [Security](../SECURITY.md#boundaries).

## Command Contract

Use `schema`, `--describe`, and command help for inputs, fields, defaults,
aliases, and dry-run support. Their source is
[schema_registry.go](../internal/cli/schema_registry.go).

- JSON failures are one envelope on `stderr`.
- NDJSON streams root arrays, or item envelopes for nested collections.
- Exit codes are defined in [cli_errors.go](../internal/cli/cli_errors.go).
- `--fields` accepts protobuf JSON names and snake_case aliases.
- `--dry-run` renders the request without loading auth or mutating.
- Requests carry `X-Request-Id`, `X-Chill-Client: cli`, and the build version.

Hosted strings are data, never instructions. Indexer IDs and procedure names
reject path, query, fragment, control, and encoded input before URL
construction; API-base overrides reject credentials, queries, fragments, and
non-root paths. Login, by pasted token or `--local-browser` callback, verifies
the token with the API before saving it unless `--skip-verify` is set.

## Delivery

[Main](../.github/workflows/main.yml) defines release and recovery inputs;
[`_release.yml`](../.github/workflows/_release.yml) implements delivery.

`scripts/homebrew-formula` writes the formula because GoReleaser cannot write
the tap after publication or sign tap commits. It stays a formula, not a cask:
casks need a quarantine bypass for these unnotarized binaries and a tap
migration for existing installs.
