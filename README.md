# chill-cli

![chill.institute cli](https://chill.institute/banner.png)

`chilly` is the terminal client for [chill.institute](https://chill.institute):
readable output for humans, stable JSON or NDJSON for scripts and agents.

## Install

```bash
brew install chill-institute/tap/chilly
npm install -g @chill-institute/cli
curl -fsSL https://raw.githubusercontent.com/chill-institute/chill-cli/main/scripts/install.sh | bash
```

Pick one, then check it with `chilly version`.

## Use

```bash
chilly auth login        # prints the token page, then reads the token from a hidden prompt
chilly search --query "dune"
chilly tv-shows --source hulu --output json
chilly add-transfer --url "magnet:?xt=urn:btih:..." --dry-run --output json
chilly schema command search --output json
```

For automation, pass `--output json`, parse only `stdout`, narrow with
`--fields`, and preview mutations with `--dry-run`. `schema` and `--describe`
report the installed command contract. Agents start with the bundled
[chilly skill](./skills/chilly-cli/SKILL.md).

## Develop

See [CONTRIBUTING.md](./CONTRIBUTING.md). Dev builds use an isolated `dev`
profile; releases use `default`.

[Architecture](./docs/architecture.md) · [Security](./SECURITY.md) ·
[MIT License](./LICENSE)
