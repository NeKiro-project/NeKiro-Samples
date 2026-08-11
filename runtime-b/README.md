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
RUNTIME_B_INSTANCE_ID
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
# Explicit registration lifecycle: set mode to `disabled` for a static sample,
# or set mode to `nacos` and provide every Nacos variable below.
RUNTIME_B_REGISTRATION_MODE
RUNTIME_B_AGENT_CARD_VERSION
RUNTIME_B_RELEASE_ID
RUNTIME_B_CARD_DIGEST
RUNTIME_B_CANONICAL_ENDPOINT
RUNTIME_B_AUDIENCE
RUNTIME_B_NACOS_API_ORIGIN
RUNTIME_B_NACOS_NAMESPACE_ID
RUNTIME_B_NACOS_GROUP_NAME
RUNTIME_B_NACOS_SERVICE_NAME
RUNTIME_B_NACOS_CLUSTER_NAME
RUNTIME_B_NACOS_PORT_NAME
RUNTIME_B_NACOS_ADVERTISED_IP
RUNTIME_B_NACOS_ADVERTISED_PORT
RUNTIME_B_NACOS_WEIGHT
RUNTIME_B_NACOS_HEARTBEAT_INTERVAL_MS
RUNTIME_B_NACOS_HEARTBEAT_TIMEOUT_MS
RUNTIME_B_NACOS_IP_DELETE_TIMEOUT_MS
RUNTIME_B_NACOS_REQUEST_TIMEOUT_MS
RUNTIME_B_NACOS_AUTH_MODE
RUNTIME_B_NACOS_ACCESS_TOKEN
RUNTIME_B_NACOS_TLS_CA_FILE
RUNTIME_B_NACOS_TLS_SERVER_NAME
RUNTIME_B_NACOS_TLS_CLIENT_CERT_FILE
RUNTIME_B_NACOS_TLS_CLIENT_KEY_FILE
```

All values are required and validated. Credentials have no default and must
not be logged, trimmed, returned in A2A payloads, or stored in platform facts.
`RUNTIME_B_INSTANCE_ID` is a non-sensitive deployment identifier included in
the sample's JSON and SSE results so Stack acceptance can prove which replica
handled an Invocation. It does not change the Agent ID, Release identity,
Router credential audience, or nested-call authorization.

With `nacos` registration, Runtime B composes Core's provider-neutral
`InstanceRegistrar` and `InstanceLease` contracts through the public SDK
`agent/registration/nacos` package. The ready instance is bound to one exact
Agent Card/Release target before serving, its freshness values are explicitly
published, and shutdown closes the lease and deregisters it. A
failed initial registration fails startup. A terminal heartbeat failure closes
the lease, makes `/readyz` return `503`, and stops serving; there is no retry,
alternate Nacos endpoint, stale lease, or Release fallback.
`RUNTIME_B_NACOS_ACCESS_TOKEN` is required only for `access_token` mode and is
never logged.

The Nacos API origin scheme is the explicit transport boundary. `http` permits
controlled plaintext only when all four TLS fields are absent. `https`
requires a private CA file and exact TLS server name; a client certificate and
key are optional only as a complete mTLS pair. The client uses TLS 1.2 or
later, never uses system roots, proxy discovery, insecure verification,
redirects, or downgrade, and reads each TLS file from a clean absolute regular
path with a 1 MiB limit. Validation and startup failures never expose a file
path, PEM block, private key, or file content.

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
isolation, Router-only nested lineage, Provider-side cancellation observation,
and a readiness request that creates no task state. The deterministic
`cancel-observed` fixture consumes a non-sensitive marker and returns only a
boolean and Provider-observed request count, including rejected duplicate
attempts. Task access is scoped by the authenticated Workspace, exact Release
provenance, capability, and Invocation; observations use the same scope without
the Invocation so a later managed call can read them. The fixture exists so
Stack can prove exactly one Router `tasks/cancel` reached this Provider without
exposing a direct Provider inspection path.

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
