# Contracts Reference

Use this reference when you need runtime truth about the current CLI surface.

## Rules

- Use `schema` or `--describe` when a command shape is uncertain.
- Use `--fields` to trim metadata and diagnostics before bringing them into context.
- Prefer command metadata over stale docs when they disagree.
- Parse only `stdout`

## Contract Discovery

- Full schema registry: `chilly schema --output json`
- Narrow schema registry: `chilly schema --fields commands.id,procedures.id --output json`
- One command: `chilly schema command search --output json`
- Narrow one command: `chilly schema command search --fields id,linked_procedure,inputs --output json`
- One procedure: `chilly schema procedure chill.v4.UserService/Search --output json`
- One output type: `chilly schema type chill.v4.ReleaseInfo --output json`
- Describe without executing: `chilly search --describe --output json`
- Discover TV source input: `chilly schema command tv-shows --fields inputs.name,inputs.description --output json`
- When command metadata includes `input_modes`, treat each mode as an either/or path. Do not treat flat `inputs[].required` as a substitute for mode-specific `required` lists.
- Type metadata includes protobuf field names and protobuf JSON names. For field masks, snake_case protobuf names such as `results.release_info.bit_depth` are accepted as aliases for protobuf JSON names such as `results.releaseInfo.bitDepth`.
- Local contract drift check: `mise run contracts:check`

## Local Diagnostics

- Full diagnostics: `chilly doctor --output json`
- Narrow diagnostics: `chilly doctor --fields auth.status,config.profile,config.api_base_url --output json`
- Config path: `chilly settings path --fields path --output json`
- Local config summary: `chilly settings show --fields profile,api_base_url,auth_token --output json`
- Current host only: `chilly settings get api-base-url --fields value --output json`
- Build metadata: `chilly version --fields version,commit,build_date --output json`
