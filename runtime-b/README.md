# Runtime B (direct `a2a-go`)

Runtime B demonstrates a direct A2A server that does not use an Agent runtime
framework. It implements the active message, streaming, task, history, and
cancellation behavior with `a2a-go`, validates Router-issued credentials with
the public NeKiro SDK, and performs managed nested calls only through the
Router.

Runtime B never imports Runtime A, Core service internals, or a database. It
does not accept a direct target Agent endpoint and does not retry or choose an
alternate route.

## Required configuration

```text
RUNTIME_B_LISTEN_ADDR
RUNTIME_B_AGENT_ID
RUNTIME_B_ROUTER_URL
RUNTIME_B_ROUTER_TOKEN
RUNTIME_B_TARGET_AGENT_ID
RUNTIME_B_TARGET_CAPABILITY
RUNTIME_B_RESPONSE_LIMIT_BYTES
RUNTIME_B_EVENT_LIMIT_BYTES
NEKIRO_AGENT_CHALLENGE_DIRECTORY
NEKIRO_AGENT_ROUTER_ISSUER
NEKIRO_AGENT_ROUTER_AUDIENCE
NEKIRO_AGENT_ROUTER_KEY_ID
NEKIRO_AGENT_ROUTER_PUBLIC_KEY_BASE64URL
```

All values are required and validated. Credentials have no default and must
not be logged, trimmed, returned in A2A payloads, or stored in platform facts.

## Test Runtime B

From the Samples repository root:

```text
go test -count=1 ./runtime-b/... ./internal/challengeproof/...
go test -race ./runtime-b/... ./internal/challengeproof/...
go vet ./runtime-b/... ./internal/challengeproof/...
docker build -f runtime-b/Dockerfile -t nekiro-runtime-b:test .
```

Success means every package prints `ok`, the race detector and vet exit with
code `0`, and the image builds. Required tests prove official A2A client
interoperability, strict one-line SSE frames, deterministic message and stream
results, task history bounds, same-task cancellation, concurrent identity
isolation, Router-only nested lineage, and a readiness request that creates no
task state.

## Run Runtime B

Build the binary with:

```text
go build -o bin/runtime-b ./runtime-b/cmd/runtime-b
```

Use NeKiro-Stack for a live run so every required Router, credential, Card,
publication, and installation setting is explicit. HTTP `200` from `/readyz`
proves process readiness. Full sample success requires the Stack backend test
to invoke Runtime B through the Router and observe committed Invocation
lineage; a direct HTTP call that bypasses Core is not acceptance.
