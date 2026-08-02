# chill-cli

![chill.institute cli](https://chill.institute/banner.png)

CLI client for [chill.institute](https://chill.institute), your favorite [put.io](https://put.io) extension since 2018.

The installed command is `chilly`, built for both humans and agents: interactive terminal output for normal use, JSON/NDJSON contracts for scripts, and dry-run support before mutations.

## Install

Homebrew:

```bash
brew install chill-institute/tap/chilly
chilly version
```

npm:

```bash
npm install -g @chill-institute/cli
chilly version
```

Install script:

```bash
curl -fsSL https://raw.githubusercontent.com/chill-institute/chill-cli/main/scripts/install.sh | bash
chilly version
```

## Quick Start

```bash
chilly auth login
chilly doctor --output json
chilly whoami --output json
chilly search --query "dune"
```

The default login flow prints the hosted web token page, then waits for the setup token to be pasted into a hidden terminal prompt. Open the printed page in a signed-in browser. If you already have a setup token:

```bash
chilly auth login --token <token>
```

`--token` is convenient for automation, but command arguments may be visible in shell history or process inspection. Prefer interactive login for humans or `--json @-` fed by a secret manager for unattended workflows.

## Scriptable Usage

Prefer explicit output formats and narrow fields when another program will read the result:

```bash
chilly search --query "dune" --fields results.title,results.release_info.bit_depth --output ndjson
chilly version --fields version --output json
chilly add-transfer --url "magnet:?xt=urn:btih:..." --dry-run --output json
printf '{"url":"magnet:?xt=urn:btih:..."}' | chilly add-transfer --json @- --dry-run --output json
printf '{"key":"api-base-url","value":"https://api.chill.institute"}' | chilly settings set --json @- --dry-run --output json
chilly schema command search --fields id,linked_procedure --output json
chilly schema type chill.v4.ReleaseInfo --fields fields.name,fields.json_name --output json
```

When `stdout` is not a TTY, `chilly` defaults to compact JSON for command results unless `--output` is set explicitly.

## Agent Prompt

Use this prompt when handing the CLI to an agent:

```text
Use `chilly` to interact with chill.institute from the terminal

Repository:
https://github.com/chill-institute/chill-cli

Read and follow this usage skill before operating the CLI:
https://raw.githubusercontent.com/chill-institute/chill-cli/main/skills/chilly-cli/SKILL.md

When only one workflow is relevant, follow the progressive-disclosure references linked from that root skill instead of loading unrelated guidance.

If `chilly` is not already on PATH, install it by following the repo README:
https://github.com/chill-institute/chill-cli/blob/main/README.md

After install, run:
chilly doctor --output json

If auth is missing, start the hosted web token flow:
chilly auth login

The command prints this page, asks the user to copy the setup token, and waits for it to be pasted back:
https://chill.institute/auth/cli-token

If you already have the token and want a non-interactive path, use:
chilly auth login --token <token>

After setup, continue with the requested task instead of stopping after install or doctor output
```

Treat the agent as an untrusted operator: prefer `--output json`, parse only `stdout`, use `--fields` to narrow reads, and use `--dry-run` before mutations.

## Develop

```bash
mise install
mise run hooks
go build ./cmd/chilly
go run ./cmd/chilly version --output json
mise run verify
```

Released binaries use the `default` profile. Dev builds default to `dev` so source runs do not reuse production config by accident.

## Docs

- [Architecture](./docs/ARCHITECTURE.md): command shape, transport, config, skills, and release flow
- [Security](./SECURITY.md): local credential and network safety notes

## Contributing

Please read the [contributing guide](./CONTRIBUTING.md).

## License

Licensed under the [MIT License](./LICENSE).
