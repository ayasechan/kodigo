# kodi — Kodi JSON-RPC Go client

Typed Go client for [Kodi's JSON-RPC API](https://kodi.wiki/view/JSON-RPC_API).
Each supported API major version gets its own generated package sharing one
transport, so one binary can talk to old and new Kodi servers.

## Setup

Install [mise](https://mise.jdx.dev/), then:

```sh
export PATH="$HOME/.local/bin:$PATH"
mise install     # installs the pinned Go toolchain, project-local
mise run check   # build + vet + test, everything green means you're good
```

No global Go installation needed; the version lives in `mise.toml`.

## Quickstart

```go
import (
    "github.com/ayasechan/kodigo"
    "github.com/ayasechan/kodigo/v13" // versioned types (kodi.Client is v13.Client)
)

c := kodi.NewClient("http://localhost:8080/jsonrpc",
    kodi.WithBasicAuth("kodi", "kodi"))

pong, err := c.JSONRPCPing(ctx)
movies, err := c.VideoLibraryGetMovies(ctx, v13.VideoLibraryGetMoviesParams{
    Properties: v13.VideoFieldsMovie{"title", "year"},
})
```

The root package always tracks the latest supported API version. To pin a
version explicitly, import it directly:

```go
import "github.com/ayasechan/kodigo/v13"

c := v13.NewClient("http://localhost:8080/jsonrpc")
```

## Talking to an unknown server

Probe first, then pick the matching versioned client — the transport is
reused, no reconnect:

```go
c, n, err := kodi.NegotiateClient(ctx, "http://localhost:8080/jsonrpc")
if err != nil {
    log.Fatal(err) // probe failure or major mismatch (see below)
}
if n.Compatibility == kodi.ServerOlderMinor {
    log.Println("old server: newer fields may be missing")
}
```

For a server on an older major, wrap the probed transport in the matching
version (same pattern as the tested `Example_versionSwitch`):

```go
base := kodi.NewClient(endpoint, kodi.WithBasicAuth("kodi", "kodi"))
n, err := kodi.Negotiate(ctx, base)
if err != nil {
    log.Fatal(err)
}
switch n.Server.Major {
case v13.APIMajor:
    c := v13.Wrap(base.Client) // *v13.Client, no reconnect
case v12.APIMajor:
    c := v12.Wrap(base.Client) // *v12.Client
default:
    log.Fatalf("unsupported Kodi API v%d", n.Server.Major)
}
// use c…
```

| server vs client | result | meaning |
| ---------------- | ------ | ------- |
| same major, server minor ≥ client | `kodi.Compatible` | full support |
| same major, server minor < client | `kodi.ServerOlderMinor` | works, newer fields may be missing |
| major differs | `kodi.IncompatibleMajor` | do not use this client version |

## Supported versions

| Package | Kodi API | Kodi | Schema | Status |
| ------- | -------- | ---- | ------ | ------ |
| `v13` (also `kodi` root) | v13 | 20 Nexus 及以上 | 13.5.0 (Omega) | supported (latest) |
| `v12` | v12 | 19 Matrix | 12.4.0 (Matrix branch) | supported |

`patch` versions are ignored (internal changes only); within a major, the
package is generated from the newest minor schema vendored under `schema/`
(sources: [`docs/schema.md`](docs/schema.md)).

## Docs

- [`docs/architecture.md`](docs/architecture.md) — package layout, dependency rules, negotiation design
- [`docs/codegen.md`](docs/codegen.md) — generator usage, schema→Go mapping, adding a new API version
- [`docs/schema.md`](docs/schema.md) — vendored schema structure the generator depends on
- [`docs/development.md`](docs/development.md) — toolchain, tasks, testing, contributor workflow
