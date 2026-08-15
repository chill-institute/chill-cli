# chill-cli

![chill.institute cli](https://chill.institute/banner.png)

`chilly` is the terminal client for [chill.institute](https://chill.institute).
It serves readable terminal output to humans and stable JSON or NDJSON to
scripts and agents.

## Install

```bash
# Homebrew
brew install chill-institute/tap/chilly

# npm
npm install -g @chill-institute/cli

# install script
curl -fsSL https://raw.githubusercontent.com/chill-institute/chill-cli/main/scripts/install.sh | bash
```

Confirm the installed build with `chilly version`.

## Sign In

```bash
chilly auth login
chilly doctor --output json
chilly whoami --output json
```

Login prints the hosted token page and waits for the token in a hidden prompt.
For controlled automation, pass JSON through stdin instead of exposing a token
in process arguments.

## Use

```bash
chilly search --query "dune"
chilly movies --output json
chilly tv-shows --source hulu --output json
chilly add-transfer --url "magnet:?xt=urn:btih:..." --dry-run --output json
```

Piped output defaults to compact JSON. For automation, set `--output json`
explicitly, parse only `stdout`, narrow large responses with `--fields`, and
preview mutations with `--dry-run`.

Discover the installed contract instead of copying command shapes from docs:

```bash
chilly schema command search --output json
chilly search --describe --output json
```

Agents should start with the bundled
[chilly skill](./skills/chilly-cli/SKILL.md), which links focused auth, read,
mutation, and contract references.

## Develop

```bash
mise install
mise run hooks
mise run verify
```

Dev builds use an isolated `dev` profile; releases use `default`.

[Architecture](./docs/ARCHITECTURE.md) · [Contributing](./CONTRIBUTING.md) ·
[Security](./SECURITY.md) · [MIT License](./LICENSE)
