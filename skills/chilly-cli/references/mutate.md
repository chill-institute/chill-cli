# Mutate Reference

Use `--json` or `--json @-` when you already know the exact request body.

## Transfers

`add-transfer` accepts `magnet:?` links and `http://` or `https://` URLs with a
host; anything else fails locally.

- Preview: `chilly add-transfer --url "magnet:?xt=..." --dry-run --output json`
- Attribute a catalog: add `--movie-source trakt` or `--tv-source netflix`.
  These flags are mutually exclusive and do not combine with `--json`; omit
  both for a direct or search send.
- Exact body: `printf '{"url":"magnet:?xt=...","catalogOrigin":{"moviesSource":"MOVIES_SOURCE_TRAKT"}}' | chilly add-transfer --json @- --dry-run --output json`

## Hosted Settings

- One field: `chilly user settings set sort-by title --dry-run --output json`
- Full replacement: pipe a complete settings object, as
  `chilly user settings get --output json` returns it, to
  `chilly user settings set --json @- --dry-run --output json`.
- Download folder: `chilly user download-folder set 42 --dry-run --output json`
  or `chilly user download-folder clear --dry-run --output json`
- Browse TV providers with `chilly tv-shows --source <provider>`; patch
  `catalog.tvShowsSource` only to manage the saved setting.

## Local and Update

- API host: `chilly settings set api-base-url https://api.chill.institute --dry-run --output json`
- Check for updates: `chilly self-update --check --output json`
- Preview an install: `chilly self-update --version v0.1.0 --dry-run --output json`
