# Security

Report vulnerabilities privately with **Report a vulnerability** in this
repository's Security tab. Do not open a public issue.

Useful reports include credential exposure, installer or updater integrity,
unsafe filesystem behavior, command injection, and private data in output or
logs. Include the affected version, impact, and a minimal reproduction.

## Boundaries

- Remote API hosts require HTTPS; loopback development may use HTTP.
- Config directories use mode `0700`; config files use `0600` and atomic writes.
- Input reaches the network only after local validation.
- Machine results use `stdout`; prompts and diagnostics use `stderr`.
- Mutations expose `--dry-run` or explicit JSON request paths where supported.

Test only accounts and data you control. Security fixes target the latest
release and `main`.

## Release Integrity

Install and self-update paths verify `checksums.txt`. GitHub Actions also
attests each released archive; npm packages wrap those same binaries.

```bash
VERSION="$(gh release view --repo chill-institute/chill-cli --json tagName -q .tagName)"
ARCHIVE="chilly_${VERSION#v}_darwin_arm64.tar.gz"

gh release download "$VERSION" --repo chill-institute/chill-cli --pattern "$ARCHIVE"
gh attestation verify "$ARCHIVE" --repo chill-institute/chill-cli
```

Adjust the archive name for your platform.
