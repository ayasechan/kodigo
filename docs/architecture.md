# Architecture

## Goals

1. **Typed access** to Kodi's JSON-RPC API, generated from the real schema —
   no hand-maintained bindings that rot.
2. **Multi-version**: one binary talks to old and new Kodi servers.
3. **Small hand-written surface**: transport and negotiation are written once;
   everything version-specific is generated and never edited by hand.

## Package map

```
schema/v12/, schema/v13/ vendored Kodi introspection schemas (JSON, source of truth)
internal/rpc/      shared, version-agnostic transport
v12/, v13/         generated typed clients per API major
<root: kodi>       thin facade over the latest version + negotiation
internal/gen/      generator (stdlib only; thin CLI in cmd/kodi-gen)
```

Dependency direction (strict DAG, no cycles):

```
v12, v13 ──imports──▶ internal/rpc
root ──imports──▶ internal/rpc, v12, v13
```

Rules:

- Version packages (`v12`, `v13`, …) may import **only**
  `internal/rpc` and the standard library. They never import the root
  package — this is what keeps the graph acyclic when the root re-exports
  them.
- The root package never contains version-specific logic beyond selecting
  the default version. All comparison logic is version-agnostic.
- Generated files (`v13/types.go`, `methods.go`, `notifications.go`,
  `client.go`) are never edited by hand; see [codegen](codegen.md).

## Transport (`internal/rpc`)

`rpc.Client` speaks JSON-RPC 2.0 over HTTP POST (Kodi webserver,
default `http://<host>:8080/jsonrpc`): request envelope with auto-increment
`id`, basic auth, typed `RPCError` on failure. `Call(ctx, method, params,
result)` takes/returns `any`, so it works for every API version.

`rpc.Caller` is the one-method interface satisfied by `*rpc.Client` **and**
by every versioned client (via promotion from the embedded `*rpc.Client`).
The probe helpers (`ProbeServerVersion`, `Negotiate`) accept `rpc.Caller`
instead of a concrete client.

## Versioned client (`v13`, …)

Each version package contains:

- `types.go`, `methods.go`, `notifications.go` — generated from its schema.
- `client.go` — generated wrapper:

```go
type Client struct {
	*rpc.Client
}
func NewClient(endpoint string, opts ...rpc.Option) *Client
func Wrap(c *rpc.Client) *Client
```

`Wrap` is the multi-version hinge: negotiate on the shared transport, then
adapt it to the selected version with no reconnect. `APIMajor`/`APIMinor`
consts expose what the package speaks.

## Root facade (`kodi.go`, `version.go`)

- `Client`/`Option`/`RPCError` are type aliases (`type X = …`), so the full
  typed method set is available without importing version packages;
  `WithBasicAuth`/`WithHTTPClient` are `var` aliases of the transport
  constructors. `NewClient` builds the default client.
- `SchemaVersion` / `SupportedAPIVersion` re-exported from the default
  version package (single source of truth, no drift).
- Negotiation layer, kept version-agnostic on purpose:
  `ProbeServerVersion(ctx, rpc.Caller)` (raw `JSONRPC.Version` call),
  pure `Check(server, client)`, `Negotiate` (probe + check against the
  default), `NegotiateClient` (build + probe in one step, 10s probe timeout
  via `DefaultNegotiateTimeout` when the context has no deadline).

## Versioning policy

- **API major → Go package** (`v13`, `v12`, …). Major bumps are breaking by
  definition, so versions never share types.
- **API minor → regenerate in place.** Same major + newer minor schema =
  re-run the generator over the old package. Additive only.
- **API patch → ignored.** Internal changes, invisible on the wire.
- **Module version ≠ API version.** The Go module follows semver
  independently; API versions live in subpackage paths. The schema source
  per version package is recorded in [`schema.md`](schema.md).

## Adding a version

Checklist (details in [codegen](codegen.md)):

1. Vendor `schema/vXX/`.
2. `go run ./cmd/kodi-gen -schema schema/vXX -out vXX -package vXX`.
3. Hand-write `vXX/doc.go`; add smoke tests and run the audit.
4. `go build ./... && go vet ./... && go test ./...`.
5. Register in `SupportedVersions()` (`kodi.go`) and extend the version
   switch (README snippet + `example_test.go` pattern).
