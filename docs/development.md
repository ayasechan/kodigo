# Development

## Prerequisites

- [mise](https://mise.jdx.dev/) — the only thing you install by hand.
  The Go toolchain is pinned project-local in `mise.toml` (`go 1.27.1`).

```sh
export PATH="$HOME/.local/bin:$PATH"
mise install
```

All commands below assume that `PATH`; `mise run` / `mise exec` set it up
for the command automatically.

## Tasks (`mise.toml`)

| Task | Runs | When |
| ---- | ---- | ---- |
| `mise run regen` | both `kodi-gen` invocations (v13 + v12) | after schema or generator changes |
| `mise run check` | `go build ./... && go vet ./... && go test ./...` | before every commit |
| `mise run test` | `go test ./...` | quick iteration |

List them with `mise tasks`.

## Workflow

- **Schema change** (new minor): replace `schema/vXX/`, `mise run regen`,
  `mise run check`, review the diff (should be additive only).
- **Generator change**: edit `internal/gen`, `mise run regen`,
  `mise run check`, plus the [audit](codegen.md#audit-after-any-generator-change).
- **Hand-written code** (`internal/rpc`, root facade, `*/doc.go`): edit,
  `mise run check`. Never edit files headed `Code generated … DO NOT EDIT`.

## Testing

Tests are colocated by layer, all against `httptest` fakes (no Kodi needed):

| Package | File | Covers |
| ------- | ---- | ------ |
| `internal/gen` | `naming_test.go` | schema→Go naming rules (`exportName`, const names) |
| `internal/rpc` | `rpc_test.go` | transport: method/params passthrough, result decode, `RPCError` mapping |
| `v12` | `kodi_test.go` | v13 smoke-test mirror (enums, round-trip, typed call), plus `APIMajor`/`SchemaVersion` assertions |
| `v12` | `example_test.go` | runnable doc example (ping) |
| `v13` | `kodi_test.go` | enums, struct round-trips, `extends` embedding, `omitempty` policy, typed call |
| `v13` | `example_test.go` | runnable doc examples (typed calls) |
| root | `version_test.go` | version parsing, pure `Check` matrix, probe + `NegotiateClient` |
| root | `example_test.go` | runnable doc examples (negotiation, version switch) |

Conventions:

- Fake servers return canned `{"jsonrpc":"2.0","id":1,"result":…}` payloads
  (`error` envelope only in the `RPCError` case); keep them minimal — one
  fake server per test where possible (a single `TestClientCall` smoke may
  still cover Ping plus one typed call).
- `Example…` functions carry `// Output:` comments: they are documentation
  that fails the build when it lies. Prefer adding an example over a README
  snippet for any user-facing flow.
- Run the full suite with `-count=1` when verifying (`mise exec -- go test
  -count=1 ./...`); plain `mise run test` may serve cached results.

## Style

- Generated code: `gofmt`-clean by construction (`format.Source` in
  `internal/gen/gen.go`). Do not run `gofmt -w` as a substitute for fixing
  the generator.
- Hand-written code: standard `gofmt` + `go vet`; no external linters.
- JSON tags always mirror the schema exactly (`json:"songid"` even though
  the field is `SongID`); deviations break wire compat silently.
- Comments on exported identifiers; doc comments on schema-derived types
  come from the schema's `description` — improve wording upstream in the
  generator, not per-type.

## CI suggestion

Assumes a git checkout (the `git diff` gate below needs one):

```sh
mise install
mise run regen
git diff --exit-code -- v12/ v13/   # regen must be a no-op on committed code
mise run check
mise exec -- go test -count=1 ./...
```
