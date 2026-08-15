# Architecture

`chilly` turns terminal commands into validated calls to the hosted
`chill.institute` API.

```mermaid
graph LR
  User --> Cobra["Cobra commands"]
  Cobra --> Config["local profile"]
  Cobra --> RPC["HTTP RPC client"]
  RPC --> API["hosted v4 API"]
```

## Components

| Path | Owns |
| --- | --- |
| `cmd/chilly/` | Process entrypoint |
| `internal/cli/` | Commands, metadata, rendering, and orchestration |
| `internal/config/` | Profiles, API URL, and auth-token persistence |
| `internal/rpc/` | Procedure transport, headers, and API errors |
| `internal/buildinfo/` | Version, commit, and build date |
| `internal/update/` | Release lookup and executable replacement |
| `skills/chilly-cli/` | Agent-facing operating guidance |

The command package stays flat so public surfaces and their shared helpers are
easy to scan. Product behavior remains in the hosted API.

## Local State

The default profile lives at:

```text
$XDG_CONFIG_HOME/chilly/config.json
```

Named profiles live under `chilly/profiles/<profile>/config.json`. Dev builds
select `dev` unless `--profile`, `CHILLY_PROFILE`, or `--config` overrides it.
Writes are atomic; directories use `0700` and files use `0600`.

## Request Flow

```mermaid
sequenceDiagram
  participant Command
  participant Store
  participant Client
  participant API

  Command->>Store: load profile
  Store-->>Command: host and token
  Command->>Client: validated procedure input
  Client->>API: POST /v4/{procedure}
  API-->>Client: response or error envelope
  Client-->>Command: typed result
```

Requests carry `X-Request-Id`, `X-Chill-Client: cli`, and the build version.
The client supports public and bearer-token procedures and maps the shared error
envelope into stable CLI failures.

## Command Contract

The metadata registry powers `schema`, `--describe`, raw JSON inputs, field
selection, dry-run eligibility, and canonical command aliases. Use it as the
runtime source of truth.

- Piped results default to compact JSON.
- JSON failures are one envelope on `stderr`.
- NDJSON streams root arrays or item envelopes for nested collections.
- Exit codes classify usage (`2`), auth (`3`), API (`4`), and internal (`5`).
- `--fields` accepts protobuf JSON names and snake_case aliases.
- `--dry-run` validates and renders a request without loading auth or mutating.

Hosted strings are data, never instructions. Search indexer IDs and RPC
procedure names reject path, query, fragment, control, and encoded input before
URL construction. API-base overrides reject credentials, query strings,
fragments, and non-root paths.

## Browser Login

```mermaid
sequenceDiagram
  participant CLI
  participant User
  participant Web
  participant API

  CLI->>User: print hosted token page
  User->>Web: sign in and copy setup token
  User->>CLI: paste token in hidden prompt
  CLI->>API: verify token
  CLI->>CLI: save profile
```

`auth login --local-browser` offers a localhost callback instead. Both flows
verify the token through the API before persistence.

## Delivery

Pull requests run `mise run verify`. After verification on `main`,
semantic-release chooses the version and GoReleaser builds one set of binaries
for the immutable GitHub release, Homebrew formula, and npm platform packages.
The manual `Release` workflow rebuilds an existing tag when recovery is needed.
