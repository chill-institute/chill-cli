# Contracts Reference

## Discovery

- Registry: `chilly schema --fields commands.id,procedures.id --output json`
- One command: `chilly schema command search --fields id,linked_procedure,inputs --output json`
- One procedure: `chilly schema procedure chill.v4.UserService/Search --output json`
- One output type: `chilly schema type chill.v4.ReleaseInfo --output json`
- Describe without running: `chilly search --describe --output json`

When a command lists `input_modes`, each mode is an alternative with its own
`required` list; flat `inputs[].required` does not replace it. Field masks
accept protobuf JSON names (`results.releaseInfo.bitDepth`) and snake_case
aliases (`results.release_info.bit_depth`).

## Diagnostics

- Everything: `chilly doctor --output json`; narrow auth and host checks are
  in [Auth](./auth.md)
- Build: `chilly version --fields version,commit,build_date --output json`
