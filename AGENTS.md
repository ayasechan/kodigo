# AGENTS.md

## Toolchain

- `go` is NOT on PATH. Always prefix: `export PATH="$HOME/.local/bin:$PATH"`,
  then `mise exec -- go …` or `mise run …`. Bare `go` fails.
- Pinned Go lives in `mise.toml` (`go 1.27.1`, project-local). Never install
  another toolchain.

## Commands

- `mise run check` — build + vet + test everything. Run before finishing.
- `mise run test` — may serve cached results; use
  `mise exec -- go test -count=1 ./...` when verifying.
- `mise run regen` — regenerates **both** version packages (v13 + v12).
  Run after touching `schema/` or `internal/gen`, then `mise run check`.

## Generated code — do not touch

- `v12/` and `v13/` (`types|methods|notifications|client.go`, header
  `Code generated … DO NOT EDIT`) are generator output. Fix the bug in
  `internal/gen`, never the output.
- Generator output must be gofmt-clean by construction (`format.Source` in
  `internal/gen/gen.go`) and byte-identical across runs. After generator
  changes: snapshot md5 of `v12/*.go v13/*.go`, regen, `gofmt -l .` must be
  empty, md5-diff must show only intended changes.
- Dup audit after generator changes (expect zero everywhere); script in
  `docs/codegen.md` — run it for **each** version package.

## Package boundaries (DAG, no cycles)

- `v12`, `v13` import **only** `internal/rpc` + stdlib. Never import root.
- Root (`kodi.go`, `version.go`) imports `internal/rpc` + `v12` + `v13`.
- `internal/*` is invisible to external modules: user-facing examples must
  use public API only (`kodi.NewClient` + `vXX.Wrap(base.Client)`), never
  `github.com/ayasechan/kodigo/internal/…` imports.

## Tests

- All fakes via `httptest`, no live Kodi needed. `Example…` funcs carry
  `// Output:` — they are compile-checked docs; keep them passing when
  changing public API.

## Schema

- Vendored snapshots: `schema/v13` = Omega 13.5.0, `schema/v12` = Matrix
  tip 12.4.0. Back up `schema/vXX/` to `/tmp` before replacing.
- Version numbers referenced in docs/tests must be re-derived after a
  schema swap (counts, delta lists, fake versions in examples).

## Docs (keep in sync with code)

- `README.md` user-facing; `docs/{architecture,codegen,schema,development}.md`
  developer-facing. When changing behavior, update the matching doc in the
  same pass — especially `docs/schema.md` numbers and the codegen mapping
  table.
