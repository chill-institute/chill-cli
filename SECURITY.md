# Security

If you believe you have found a security issue in the `chilly` CLI, please report it privately.

## Contact

- email: [chillardinho@chill.institute](mailto:chillardinho@chill.institute)

Private reports are preferred for security issues.

If you are unsure whether something is sensitive, feel free to email first.

## Scope

Useful reports include issues involving:

- auth token exposure or unsafe local credential handling
- updater, installer, or release integrity bypasses
- privilege escalation through config, profiles, or filesystem behavior
- command injection, path traversal, or unsafe external input handling
- private data exposure in CLI output, logs, or config files

## Security Posture

`chilly` is designed with the assumption that the agent is not a trusted operator.

- opaque IDs, base URLs, and procedure names should be validated locally before use
- API endpoints must use HTTPS except for explicit loopback development hosts
- machine-readable output belongs on `stdout`; prompts, warnings, and recovery hints belong on `stderr`
- mutating commands should offer preview paths such as `--dry-run`
- request bodies should prefer explicit JSON contracts over heuristic shell parsing when possible

## Guidelines

- test only against accounts and data you control
- avoid anything destructive, disruptive, or automated at scale
- do not target infrastructure outside the published CLI release, npm packages, Homebrew tap, or documented API flows

## Supported Versions

Security fixes are applied on a best-effort basis to the latest released CLI version and the latest code on `main`

## Release Integrity

- published archives include `checksums.txt`
- `scripts/install.sh` verifies archive checksums before installation
- `chilly self-update` verifies archive checksums before replacing the current executable
- npm packages install the same GoReleaser-built binaries through the `@chill-institute/cli` launcher
- GitHub Actions publishes release artifact attestations for released archives

## Verify A Release

```bash
VERSION="$(gh release view --repo chill-institute/chill-cli --json tagName -q .tagName)"
ARCHIVE="chilly_${VERSION#v}_darwin_arm64.tar.gz"

gh release download "$VERSION" --repo chill-institute/chill-cli --pattern "$ARCHIVE"
gh attestation verify "$ARCHIVE" --repo chill-institute/chill-cli
```

Adjust the archive name for your platform when needed.

## Disclosure

Please give us a reasonable chance to investigate and fix the issue before sharing details publicly.

There is no bug bounty program right now, but thoughtful reports are appreciated.
