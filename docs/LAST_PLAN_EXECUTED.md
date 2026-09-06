# PLAN — ddlc: post-split hardening (docs, TUI handler, consumer contract)

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Born from the `webtyp/orm` repo split (2026-07-10). Self-contained: the
> executing agent has zero prior context.

## Prerequisite (run first)

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

Run all tests with `gotest` (never plain `go test`).

## Context (zero-context summary)

`webtyp.com/ddlc` was just split out of `webtyp.com/orm`.
It is the **DDL contract leaf** of the webtyp SQL ecosystem and must stay a
leaf: its root module depends ONLY on `webtyp/model` and `webtyp/fmt`.

It owns three things (already moved, compiling, tests green):

- `Exporter` (`exporter.go`) — implemented by the SQL compilers
  (`webtyp/sqlt`, `webtyp/postgres`): `ExportDDL(models []model.Model)
  (string, error)` returns CREATE TABLE + indexes in FK dependency order.
- `TopologicalSort` (`sort.go`) — Kahn's BFS over FK references; models
  expose them via `SchemaExt() []FieldExt`.
- `FieldExt` (`field_ext.go`) — FK metadata (`Ref`, `RefColumn`,
  `OnDelete`), **moved here from the orm runtime** so sqlt/postgres and
  generated code no longer need the orm module for DDL concerns.
- `cmd/ddlc` — CLI (own go.mod, may depend on ormc/sqlt/postgres; the root
  module must NOT).

Consumers and their migration (tracked in their own repos' plans, listed
here so this repo's docs describe the contract correctly):

| Consumer | Change |
|---|---|
| `webtyp/sqlt`, `webtyp/postgres` | import `webtyp.com/orm/ddl` → `webtyp.com/ddlc`; `orm.FieldExt` → `ddlc.FieldExt` |
| `webtyp/ormc` (generator) | generated `SchemaExt()` emits `[]ddlc.FieldExt` (its plan, stage 0) |
| `webtyp/sqlmcp` | already imports `ddlc` (done in split) |
| `webtyp/app` | consumes ddlc through its own TUI handler (stage 2 below) |

**Ecosystem rules:** no stdlib in WASM-shared code (`webtyp/fmt`), no
`any`/`map` in public APIs, typed constants, errors propagate, `gotest` only.

## Stage 1 — documentation

- `README.md`: replace the gonew stub. Document: what ddlc is (DDL export
  contract + FK topological sort + `FieldExt`), who implements `Exporter`
  (sqlt/postgres), who calls it (ormc's `ExportSQL`, sqlmcp's
  `db_export_schema`, the `cmd/ddlc` CLI), and the leaf-dependency guarantee.
- `docs/ARCHITECTURE.md` (create): why the split exists (orm updates no
  longer force sqlt/postgres/sqlmcp version bumps), the dependency diagram
  (mermaid): `sqlt/postgres → ddlc ← ormc`, `cmd/ddlc → ormc+sqlt+postgres`
  (separate module), and why `FieldExt` lives here and not in `model`
  (it is DDL/FK metadata, meaningless outside SQL adapters).
- `cmd/ddlc/README.md`: verify flags/examples still match `main.go` after
  the import rewrite.

## Stage 2 — TUI handler for `webtyp/app` (zero coupling)

`webtyp/app` will consume ormc and ddlc **separately, each with its own
TUI handler**. ormc already has one (`Name() "ORMC"` in its `handler.go`).
ddlc needs its own so DDL export is a first-class dev-console action.

Create `handler.go` (root package):

- `type Handler struct` with `New() *Handler`.
- `Name() string` → `"DDLC"` (typed constant).
- `SetLog(fn func(messages ...any))` — same pattern as ormc's handler; nil
  log fn = silently discarded messages is FORBIDDEN, default to no-op but
  errors from Execute must still propagate to the caller.
- The handler does NOT scan Go source (that is ormc's job) and does NOT
  import ormc/sqlt/postgres (root stays leaf). It receives the export
  operation injected:

```go
// ExportFunc produces the full DDL for the current project.
// webtyp/app wires it to ormc.ExportSQL + the dialect compiler.
type ExportFunc func() (sql string, err error)

func (h *Handler) SetExport(fn ExportFunc)
```

- `Execute() error` (or the devtui execution contract app uses — check
  `webtyp/devtui` interfaces and satisfy them **structurally**, never
  importing devtui): runs the injected func, logs the result destination,
  propagates errors. Missing `SetExport` → explicit error (`ddlc handler:
  export function not configured`), never a nil-func panic.
- Typed constants for every message/prefix. No repeated string literals.

Tests (`gotest`): handler with stub ExportFunc returns SQL; handler without
ExportFunc errors with the configured message; error from ExportFunc
propagates verbatim.

## Stage 3 — CI hygiene

- `cmd/ddlc/go.mod` currently uses local `replace` directives
  (`ddlc => ../..`, `ormc => ../../../ormc`, `model` pinned v0.0.6). Keep
  them until ddlc/ormc are published, then the maintainer drops them at
  publish time — leave a `// TODO(publish)` comment on each replace line.
- Root `go.mod` must keep ONLY `model` + `fmt` as direct deps. Add a test or
  doc note stating the leaf guarantee so future changes don't silently grow
  the graph.

## Acceptance criteria

1. `gotest ./...` green (root module and `cmd/ddlc` build).
2. Root `go.mod` direct deps: exactly `webtyp/model`, `webtyp/fmt`.
3. `Handler` satisfies app's TUI contract structurally (no devtui import);
   unset ExportFunc errors explicitly.
4. README + ARCHITECTURE written as specified; no gonew stub text remains.

## Stages

| Stage | File(s) | Action |
|---|---|---|
| 1 | `README.md`, `docs/ARCHITECTURE.md`, `cmd/ddlc/README.md` | contract + split rationale |
| 2 | `handler.go`, `handler_test.go` | injected-export TUI handler "DDLC" |
| 3 | `go.mod`, `cmd/ddlc/go.mod` | replace hygiene + leaf guarantee |
