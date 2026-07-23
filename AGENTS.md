# CLI

`cli` is the `chill.institute` command-line client. Treat it as an agent-first SDK surface with a human CLI on top.

## Stack

- Go with Cobra for commands
- manual HTTP RPC client for the hosted API
- local config store for auth token and API base URL
- `mise` for shared repo tasks

## Commands

- `go build ./cmd/chilly`
- `go run ./cmd/chilly version --output json`
- `go test ./...`
- `mise run smoke`
- `mise run verify`
- `mise run fmt`
- `mise run coverage:report`

For command-surface changes, also run:

- `go run ./cmd/chilly <command> --help`

## Worktrees

- Managed Codex and Claude worktrees copy the optional hosted-integration `.env.local` through `.worktreeinclude`; `mise run test:integration` loads it automatically.
- Standard development needs no secrets. Obtain live user tokens through the normal CLI authentication flow, not the shared Infisical development inventory.

## Conventions

- Prefer machine-readable contracts first. New behavior should have a stable JSON story before nicer human formatting.
- When `stdout` is not a TTY and `--output` is not set explicitly, command results default to JSON. Keep that contract stable for agent workflows.
- Treat the agent as an untrusted operator. New surfaces should validate opaque IDs and base URLs locally before they reach the network or filesystem.
- Keep `stdout` for command results and `stderr` for prompts, progress, warnings, and recovery hints.
- Keep auth requirements, flags, and schema/describe surfaces explicit.
- Treat indexer health as a tri-state contract: `healthy`, `degraded`, and `down`. Do not collapse partially working providers into binary healthy/down wording in new CLI surfaces or docs.
- Git hooks remain as tiny executable launchers because Git requires hook files, but their behavior should stay in `mise.toml`
- If a command surface, auth flow, default, or output contract changes, update the user-facing chilly skill library in [skills/](./skills/) in the same pass.

## Read More

- command and transport architecture: [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)
- security posture: [SECURITY.md](./SECURITY.md)
- setup, release flow, and validation: [CONTRIBUTING.md](./CONTRIBUTING.md)
