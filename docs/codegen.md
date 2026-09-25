# Code generation

`internal/gen` (thin CLI: `cmd/kodi-gen`) reads a Kodi introspection schema
directory and emits a versioned client package. Stdlib only
(`encoding/json`, `go/format`). Schema layout: [`schema.md`](schema.md).

## Usage

```sh
go run ./cmd/kodi-gen -schema schema/v13 -out v13 -package v13
# or: mise run regen (runs the above for every vendored version)
```

| Flag | Default | Meaning |
| ---- | ------- | ------- |
| `-schema` | `schema` | schema dir (`types.json`, `methods.json`, `notifications.json`, `version.txt`) |
| `-out` | `.` | output dir (created if missing) |
| `-package` | base of `-out` (`"."` → `"kodi"`) | Go package name of emitted files |
| `-rpcimport` | `github.com/ayasechan/kodigo/internal/rpc` | import path of the shared transport (package name must be `rpc`) |

Outputs in `-out`: `types.go`, `methods.go`, `notifications.go`, `client.go`
(Client wrapper, `NewClient`/`Wrap`, `APIMajor`/`APIMinor`).

Guarantees:

- **Deterministic**: top-level key order follows the schema files (token
  stream, not Go maps); struct fields and stub lists are sorted. Running
  twice yields byte-identical files.
- **gofmt-clean**: output goes through `format.Source`; a dirty `gofmt -l`
  after regen means a generator bug, not a formatting step you forgot.

## Schema → Go mapping

| Schema | Go |
| ------ | -- |
| Named type `Player.Repeat` | `type PlayerRepeat …` (dots → CamelCase) |
| JSON field `songid`, `channeluid` | `SongID`, `ChannelUID` (ID/UID/URL/URI/TV/PVR/… initialisms; tags keep original names) |
| `{"type":"string","enum":[…]}` | `type X string` + typed consts (`PlayerRepeatOff = "off"`) |
| `{"type":"integer"/"number"/"boolean"}` | `int` / `float64` / `bool` (`Integer` accepted as `integer`) |
| `{"type":"object","properties":{…}}` | `struct` |
| `{"type":"array","items":…}` | slice |
| `{"extends":"Base","properties":{…}}` | struct embedding `Base` |
| `{"extends":…,"items":…}` (array subtype) | slice type, extends noted in doc |
| `$ref: "Foo"` (or `"type": "Video.Ratings"` ref-by-name) | `Foo` Go type |
| `["null", T]` in anonymous position | `*T` (named top-level nullable types stay value types, e.g. `OptionalBoolean bool`) |
| Other unions (`Setting.Value`, `Playlist.Item`, …) | `any`, with a doc comment pointing at the schema key |
| Method `Player.Open` | `MethodPlayerOpen` const, `PlayerOpenParams` struct, `PlayerOpenResult` (or inline primitive), `func (c *Client) PlayerOpen(…)` |
| Notification `Player.OnPlay` | `NotificationPlayerOnPlay` const, `PlayerOnPlayData` type, `…Handler` func type |

Nullability / optionality (struct fields):

- `required: true` → plain field, no `omitempty`.
- Optional → `json:",omitempty"`, plus `*T` when `T` is a scalar, string-enum
  alias, or struct (slices/maps/`any` are already nil-able).
- An optional field's `default` becomes a `// Default: …` doc line; `description` becomes the
  doc comment. Only named string-enum types get typed consts — inline
  (unnamed) enums on fields collapse to plain `string`.

Anonymous inline objects/arrays-of-objects inside params/results/data get
stable generated names (`PlayerOpenParamsOptions`,
`PlayerGetActivePlayersResultItem`, …) instead of anonymous structs, so
godoc stays readable.

## Unresolved `$ref`s

Some refs (currently 14 per version, e.g. `List.Filter.Fields.Movies`,
`Addon.Types`, `List.Filter.Operators`) are absent from `types.json`. The
generator emits them as documented `string` stubs and prints the list on
any run where some remain:

```
kodi-gen: 14 unresolved $refs emitted as string stubs: …
```

Treat that line as a to-do list when vendoring a new schema: if a newer
schema defines them, the stubs disappear on regen. Affected params/results
stay usable as `any`/`string`.

## Adding a new API version

`v12` (Matrix branch, schema 12.4.0) is the worked example — follow the same steps:

1. Vendor the schema into `schema/vXX/` (from Kodi source
   `xbmc/interfaces/json-rpc/schema/` at the matching release tag, or via
   `JSONRPC.Introspect` against a running server). Keep `version.txt` with
   the `JSONRPC_VERSION x.y.z` line.
2. Generate: `go run ./cmd/kodi-gen -schema schema/vXX -out vXX -package vXX`.
3. Hand-write `vXX/doc.go` (copy `v13/doc.go`, adjust version).
4. Add smoke tests: at minimum a round-trip test of 2–3 central types and
   one fake-server method call (see `v13/kodi_test.go`); run the audit below.
5. `go build ./... && go vet ./... && go test ./...`.
6. Register the version in `SupportedVersions()` (`kodi.go`) and extend the
   documented version switch (README + `example_test.go` pattern).

## Audit (after any generator change)

```sh
python3 - <<'EOF'
import re
from collections import Counter
src = open('v13/types.go').read() + open('v13/methods.go').read() \
    + open('v13/notifications.go').read() + open('v13/client.go').read()
types = re.findall(r'^type (\w+)', src, re.M)
consts = re.findall(r'^\t(\w+)\s+(?:[\w\[\]\*]+\s+)?=\s*"', src, re.M)
funcs = re.findall(r'^func \(c \*Client\) (\w+)\(', src, re.M)
print('types:', len(types), 'dup:', [k for k, v in Counter(types).items() if v > 1])
print('consts:', len(consts), 'dup:', [k for k, v in Counter(consts).items() if v > 1])
print('methods:', len(funcs), 'dup:', [k for k, v in Counter(funcs).items() if v > 1])
EOF
```

Expect zero duplicates everywhere. (Note: the const regex must tolerate
gofmt alignment spaces — `(\w+)\s+= "` with single-space matching
undercounts.)

## Troubleshooting

- `gofmt -l` dirty after regen → generator bug (output must already be
  formatted); fix the generator, never the output.
- New schema, fewer/more methods than expected → diff against the previous
  package; minor bumps only add, so a disappearance deserves a look at the
  schema source.
- A `$ref` you expected is now a stub → it vanished from `types.json`;
  check the vendored schema.
