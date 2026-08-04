# NeKiro Samples

This repository owns NeKiro's cross-runtime sample Agents. Both validate
Router-issued credentials with the public Go SDK, call other Agents only
through the Router, and consume language-neutral contracts from the exact Core
module revision in `go.mod`. They do not import Core service internals, access
platform databases, retry, select alternate endpoints, or carry copied
contract/SDK source.

## Sample matrix

| Sample | Runtime implementation | What it proves | Detailed guide |
|---|---|---|---|
| Runtime A | `trpc-agent-go` adapter | Framework-backed execution and one managed nested Router call | [`runtime-a/README.md`](runtime-a/README.md) |
| Runtime B | direct `a2a-go` server | Direct A2A message, stream, task, cancellation, and nested-call behavior | [`runtime-b/README.md`](runtime-b/README.md) |

The two samples deliberately do not import each other. Their shared platform
behavior is limited to public Core contracts, the public Go SDK, and the A2A
wire profile. `internal/challengeproof` contains only sample-owned endpoint
ownership proof handling.

## Repository verification

```text
go mod download
go build ./...
go test -count=1 ./runtime-a/... ./internal/challengeproof/...
go test -count=1 ./runtime-b/...
go test -race ./...
go vet ./...
docker build -f runtime-a/Dockerfile -t nekiro-runtime-a:test .
docker build -f runtime-b/Dockerfile -t nekiro-runtime-b:test .
```

Verification succeeds only when:

- Go prints `ok` for `runtime-a`, `runtime-b`, and
  `internal/challengeproof`, with no failed required tests;
- Runtime A's managed nested-call, correlation, credential-isolation, and
  readiness tests pass;
- Runtime B's official A2A client operations, strict SSE framing, task
  cancellation, nested lineage, and readiness tests pass;
- the race detector and `go vet` exit with code `0`;
- both Docker builds finish with exit code `0` and produce the requested image.

These checks prove each sample in isolation. Live success is owned by
[NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack): both images
start healthy, Runtime A and Runtime B are published and installed through
Core, a managed invocation crosses the Router, and the parent/child Invocation
lineage is queryable from the Ledger. A direct Runtime-to-Runtime shortcut is
not a successful sample run.

## Verify against a Core commit

The `Core compatibility` workflow accepts one full Core commit SHA. It
temporarily resolves that exact public module revision, then builds, tests,
race-tests, vets, and image-builds both samples. Core calls this reusable
workflow after every merge to `main`; no local `replace`, copied contract tree,
or floating branch is used.

## Configuration and live execution

Every Runtime setting is explicit. Runtime-specific configuration uses the
`RUNTIME_A_*` or `RUNTIME_B_*` names defined in each `config.go`; both also
require `NEKIRO_AGENT_CHALLENGE_DIRECTORY`, `NEKIRO_AGENT_ROUTER_ISSUER`,
`NEKIRO_AGENT_ROUTER_AUDIENCE`, `NEKIRO_AGENT_ROUTER_KEY_ID`, and
`NEKIRO_AGENT_ROUTER_PUBLIC_KEY_BASE64URL`. Missing or invalid configuration
fails startup.

Do not invent placeholder credentials simply to make a process start. Use
NeKiro-Stack for a real local run so the Router, signed credential policy,
Agent Cards, publication, installation, and Ledger are all explicit.

## Pull requests

Pull requests must identify the affected Runtime, the exact Core and SDK
revisions tested, commands run, and observable success signals. Changes to A2A
behavior must state whether message, stream, task, cancellation,
authentication, or lineage semantics changed and identify the corresponding
Stack acceptance.

## Provenance

The sample history was exported from
`NeKiro-project/NeKiro@aad73c450435a9b6c76c26cc6c525fa811b0e7ad`.
The original `agents/` tree is
`9cbc9dcf86c6fcb1203cb84c19be51af1f2c90ba`, and the history-preserving export
commit is `bf6ad75a17d0245888b0416810a771584a392675`. The source repository retains
the annotated tag `pre-repository-split-2026-08-04` for original commit and
signature provenance.

Licensed under Apache-2.0. See `LICENSE`.
