# NeKiro Samples

This repository owns NeKiro's cross-runtime sample Agents:

- `runtime-a`: a `trpc-agent-go` Runtime adapted to the platform A2A boundary.
- `runtime-b`: a direct `a2a-go` Runtime.
- `internal/challengeproof`: sample-only endpoint ownership proof handling.

Both Runtimes validate Router-issued credentials with the public NeKiro Go SDK,
call other Agents only through the Router, and consume the language-neutral
contracts from the exact core module revision in `go.mod`. They do not import
core service internals, access platform databases, retry, select alternate
endpoints, or carry copied contract/SDK source.

## Build and test

```bash
go build ./...
go test ./...
go test -race ./...
go vet ./...
docker build -f runtime-a/Dockerfile .
docker build -f runtime-b/Dockerfile .
```

Every runtime setting is explicit. Runtime-specific configuration uses the
`RUNTIME_A_*` or `RUNTIME_B_*` names defined in each `config.go`; both also
require `NEKIRO_AGENT_CHALLENGE_DIRECTORY`, `NEKIRO_AGENT_ROUTER_ISSUER`,
`NEKIRO_AGENT_ROUTER_AUDIENCE`, `NEKIRO_AGENT_ROUTER_KEY_ID`, and
`NEKIRO_AGENT_ROUTER_PUBLIC_KEY_BASE64URL`. Missing or invalid configuration
fails startup.

## Provenance

The sample history was exported from
`NeKiro-project/NeKiro@aad73c450435a9b6c76c26cc6c525fa811b0e7ad`.
The original `agents/` tree is
`9cbc9dcf86c6fcb1203cb84c19be51af1f2c90ba`, and the history-preserving export
commit is `bf6ad75a17d0245888b0416810a771584a392675`. The source repository retains
the annotated tag `pre-repository-split-2026-08-04` for original commit and
signature provenance.

Licensed under Apache-2.0. See `LICENSE`.
