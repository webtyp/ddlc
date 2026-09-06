# ddlc
<img src="docs/img/badges.svg">

`ddlc` is the **DDL contract leaf** of the webtyp SQL ecosystem. It provides the core interfaces and utilities for database schema exporting and topological sorting of foreign keys.

To keep the runtime environment lightweight and free of unnecessary dependencies (especially for WASM compilation), all build-time DDL-generation capabilities and CLI schemas are decoupled from the core runtime (`webtyp/orm`) and reside here.

## Core Features

- **`Exporter` Interface (`exporter.go`):** An interface implemented by dialect-specific SQL adapters (such as `webtyp/sqlt` and `webtyp/postgres`) to convert models to their corresponding SQL `CREATE TABLE` and index definitions in foreign key dependency order.
- **`FieldExt` Struct (`field_ext.go`):** Holds foreign key database metadata (`Ref`, `RefColumn`, `OnDelete`). By locating it in `ddlc`, dialect compilers and generated models do not need to pull in the heavier runtime ORM module.
- **`TopologicalSort` (`sort.go`):** Performs Kahn's topological sort (BFS) over models implementing schema extensions (`SchemaExt() []FieldExt`) to ensure parent tables are created before child tables.

## Implementations & Consumers

### Implementers of `Exporter`
- **`webtyp/sqlt`:** The SQLite database adapter compiler.
- **`webtyp/postgres`:** The PostgreSQL database adapter compiler.

### Callers of `Exporter`
- **`webtyp/ormc`:** The ORM code generator (via `ExportSQL`).
- **`webtyp/sqlmcp`:** The MCP db tool (via `db_export_schema`).

## Leaf-Dependency Guarantee

To guarantee portability and allow compilation in frontend or WASM environments, the root package of `webtyp.com/ddlc` depends **only** on:
- `webtyp.com/model`
- `webtyp.com/fmt`

It does **not** import `webtyp/orm` or any database adapters.

## `tui` Subpackage (dev console only)

`tui/handler.go` provides a `devtui.HandlerExecution`-compatible `Handler`
(`Name`, `Label`, `SetLog`, `Execute`) that runs an injected `ExportFunc` and
writes the resulting DDL to a file (`config/schema.sql` by default, override with
`SetOutputPath`). It lives in its own package, separate from the root
contract, so that `sqlt`/`postgres`/`ormc`/`sqlmcp` — which only need the
leaf contract — never compile file-I/O or dev-console code. Only
`webtyp/app` should import `webtyp.com/ddlc/tui`.
