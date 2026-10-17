# Read Reference

Start wide enough to find IDs, then rerun narrowly with `--fields`.

## Search

- Indexers: `chilly user indexers --fields indexers.id,indexers.name,indexers.tags,indexers.status --output json`
- One indexer: `chilly search --query "dune" --indexer-id yts --fields results.title --output json`
- Streamed: `chilly search --query "dune" --fields results.title,results.release_info.bit_depth --output ndjson`
- `--indexer-id` is opaque; values with `/`, `?`, `#`, `%`, or `..` fail
  locally. `INDEXER_STATUS_DEGRADED` means partly working.

## Catalog and Account

- Profile: `chilly whoami --fields username,email --output json`
- Movies: `chilly movies --fields movies.title --output json`
- TV shows: `chilly tv-shows --source all-providers --fields source,shows.title --output json`
- Show: `chilly tv-shows detail tt0944947 --fields show.title,seasons.seasonNumber --output json`
- Season: `chilly tv-shows season tt0944947 1 --fields season.name,episodes.name --output json`
- Season downloads: `chilly tv-shows season-downloads tt0944947 1 --fields seasonPack.title,episodes.download.title --output json`
- Transfer: `chilly get-transfer 42 --fields transfer.status,transfer.percentDone --output json`
- Settings: `chilly user settings get --fields search.sortBy,catalog.moviesSource,download.folderId --output json`
- Download folder: `chilly user download-folder --fields folder.id,folder.name --output json`
- Folder: `chilly user folder get 0 --fields parent.name,files.name --output json`

`--source` takes values listed by
`chilly schema command tv-shows --fields inputs --output json`; the response
`source` field confirms which provider answered.
