# Runtime A (`trpc-agent-go`)

Runtime A demonstrates a framework-backed Agent without making the framework
part of NeKiro Core. `trpc-agent-go` is confined to Runtime A execution;
platform calls use the public NeKiro SDK and every managed nested invocation
returns through the A2A Router.

The sample has no platform database access, no Runtime B imports, no direct
target URL, no retry/cache/alternate route, and no configuration defaults.

## Required configuration

```text
RUNTIME_A_LISTEN_ADDR
RUNTIME_A_AGENT_ID
RUNTIME_A_ROUTER_URL
RUNTIME_A_ROUTER_TOKEN
RUNTIME_A_TARGET_AGENT_ID
RUNTIME_A_TARGET_CAPABILITY
RUNTIME_A_RESPONSE_LIMIT_BYTES
RUNTIME_A_EVENT_LIMIT_BYTES
NEKIRO_AGENT_CHALLENGE_DIRECTORY
NEKIRO_AGENT_ROUTER_ISSUER
NEKIRO_AGENT_ROUTER_AUDIENCE
NEKIRO_AGENT_ROUTER_KEY_ID
NEKIRO_AGENT_ROUTER_PUBLIC_KEY_BASE64URL
```

When `RUNTIME_A_REGISTRATION_MODE=nacos`, the deployment must additionally
provide the exact target fields `RUNTIME_A_AGENT_CARD_VERSION`,
`RUNTIME_A_RELEASE_ID`, `RUNTIME_A_CARD_DIGEST`,
`RUNTIME_A_CANONICAL_ENDPOINT`, and `RUNTIME_A_AUDIENCE`; the Nacos tuple;
`RUNTIME_A_NACOS_PORT_NAME`, advertised IP/port and weight; explicit heartbeat,
heartbeat-timeout, IP-delete-timeout, and request-timeout values; and the
selected authentication mode. Runtime A composes Core's `InstanceRegistrar`
and `InstanceLease` through the public SDK `agent/registration/nacos` package,
fails startup if the initial publish fails, becomes not-ready and stops on
terminal lease failure, and explicitly deregisters on shutdown.

The `RUNTIME_A_NACOS_API_ORIGIN` scheme explicitly selects the registration
transport. An `http` origin is controlled plaintext and every Nacos TLS field
must be absent. An `https` origin requires
`RUNTIME_A_NACOS_TLS_CA_FILE` and `RUNTIME_A_NACOS_TLS_SERVER_NAME`.
Mutual TLS additionally requires the complete
`RUNTIME_A_NACOS_TLS_CLIENT_CERT_FILE` and
`RUNTIME_A_NACOS_TLS_CLIENT_KEY_FILE` pair. TLS uses only the configured
private CA, TLS 1.2 or later, and exact hostname verification; system roots,
proxy discovery, redirects, insecure verification, and HTTPS downgrade are
disabled. TLS files must be clean absolute paths to regular, non-empty files
of at most 1 MiB. Startup errors do not include paths, PEM data, key bytes, or
file contents.

`NEKIRO_AGENT_CHALLENGE_DIRECTORY` is an absolute, explicitly configured
directory used only to serve provider-owned one-time HTTP ownership proofs at
`/.well-known/nekiro/challenges/{challengeId}`. It has no default and is not a
platform secret store.

The Runtime accepts the active A2A JSON-RPC profile and exposes `GET /readyz`
without starting nested work. Test fixtures cover a deterministic echo path and
a managed nested call through the Router.

## Test Runtime A

From the Samples repository root:

```text
go test -count=1 ./runtime-a/... ./internal/challengeproof/...
go test -race ./runtime-a/... ./internal/challengeproof/...
go vet ./runtime-a/... ./internal/challengeproof/...
docker build -f runtime-a/Dockerfile -t nekiro-runtime-a:test .
```

Success means all Runtime A and challenge-proof packages print `ok`, the race
detector and vet exit with code `0`, and the Docker image builds. In
particular, the named tests must prove one Router-only nested call, stable
lineage, concurrent-call isolation, no credential copying, deterministic
result mapping, and a side-effect-free readiness endpoint.

## Run Runtime A

Build the binary with:

```text
go build -o bin/runtime-a ./runtime-a/cmd/runtime-a
```

Start it only with an explicitly configured NeKiro-Stack environment. HTTP
`200` from `/readyz` proves process readiness; complete sample success also
requires a Stack invocation and queryable Ledger lineage. Process startup by
itself is not end-to-end success.

`RUNTIME_A_ROUTER_TOKEN` is an exact credential. It must not be logged,
trimmed, copied into A2A data, or placed in platform facts.
