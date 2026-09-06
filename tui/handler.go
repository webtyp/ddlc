// Package tui provides the DevTUI execution handler for DDL export.
//
// This package is intentionally separate from the ddlc root package: the
// root stays a WASM-safe leaf contract (Exporter, FieldExt, TopologicalSort)
// consumed by sqlt/postgres/ormc/sqlmcp, while this package pulls in os/file
// I/O and the DevTUI execution contract and is only meant to be imported by
// webtyp/app's dev console.
package tui

import (
	"errors"
	"os"
	"path/filepath"

	"webtyp.com/fmt"
	"webtyp.com/tui"
)

// ExportFunc produces the DDL SQL for the current project.
type ExportFunc func() (sql string, err error)

// DefaultOutputPath is used when SetOutputPath is never called.
const DefaultOutputPath = "config/schema.sql"

// Handler implements the TUI HandlerExecution and Loggable interfaces structurally.
type Handler struct {
	logFn      func(messages ...any)
	export     ExportFunc
	outputPath string
	rootDir    string
}

// New creates a new DDLC TUI handler.
func New() *Handler {
	return &Handler{
		logFn:      func(messages ...any) {}, // default no-op
		outputPath: DefaultOutputPath,
	}
}

// Name returns the TUI handler identifier.
func (h *Handler) Name() string {
	return "DDLC"
}

// Label returns the button label for DevTUI. Includes both "SQL" and "DDL"
// (the acronym alone is opaque to devs without a SQL background) and the
// actual configured output path, so the button doubles as documentation of
// where the file lands. The path is shortened relative to rootDir (when set)
// since outputPath is absolute — see SetOutputPath / SetRootDir.
func (h *Handler) Label() string {
	path := h.outputPath
	if h.rootDir != "" {
		path = fmt.PathRelativeTo(path, h.rootDir)
	}
	return "Export SQL DDL → " + path
}

// SetLog injects the logging function.
func (h *Handler) SetLog(fn func(messages ...any)) {
	if fn != nil {
		h.logFn = fn
	} else {
		h.logFn = func(messages ...any) {}
	}
}

// SetExport injects the DDL export logic.
func (h *Handler) SetExport(fn ExportFunc) {
	h.export = fn
}

// SetOutputPath overrides the file the generated DDL is written to.
// Empty path resets to DefaultOutputPath.
func (h *Handler) SetOutputPath(path string) {
	if path == "" {
		path = DefaultOutputPath
	}
	h.outputPath = path
}

// SetRootDir tells the handler the project root, used only to shorten
// outputPath for display in Label() — outputPath itself stays absolute so
// writeOutput() is correct regardless of the daemon's own working directory
// (which does not necessarily track the project root it is currently serving).
func (h *Handler) SetRootDir(path string) {
	h.rootDir = path
}

// Log message prefix and message constants to avoid repeating string literals.
// LogOpen/LogClose come from webtyp/tui (the shared handler contract) rather
// than being redeclared here — see tui.LogOpen/tui.LogClose usage below.
const (
	MsgPrefix        = "ddlc handler: "
	MsgExporting     = "Exporting DDL schema..."
	MsgExportFailed  = "Export failed: "
	MsgExportSuccess = "Export successful. Written to: "
	MsgWriteFailed   = "Write failed: "
	MsgNoModels      = "No models found — define one in a models.go file (see webtyp.com/model). Nothing exported."
	ErrNotConfigured = "ddlc handler: export function not configured"
)

// ErrNothingToExport signals an ExportFunc found nothing to write (e.g. no
// models defined in the project). ExecuteErr shows a friendly message instead
// of the generic failure one, and — like any non-nil error — never writes a
// file. Kept decoupled from any specific ExportFunc implementation (e.g.
// webtyp/ormc): implementations that aren't ormc-based can return this
// directly; ormc-based ones go through a translation at their wiring site
// (see webtyp/app/section-build.go, which knows about both ormc's
// ErrNoModelsFound and this sentinel).
var ErrNothingToExport = fmt.Err("nothing", "to", "export")

// Execute triggers the DDL export. Implements devtui.HandlerExecution structurally.
func (h *Handler) Execute() {
	_ = h.ExecuteErr()
}

// ExecuteErr executes the export, writes the result to outputPath, and
// returns the error for test assertion.
func (h *Handler) ExecuteErr() error {
	if h.export == nil {
		err := fmt.Err(ErrNotConfigured)
		h.logFn(MsgPrefix + err.Error())
		return err
	}

	h.logFn(tui.LogOpen, MsgPrefix+MsgExporting)
	sql, err := h.export()
	if errors.Is(err, ErrNothingToExport) {
		h.logFn(tui.LogClose, MsgPrefix+MsgNoModels)
		return err
	}
	if err != nil {
		h.logFn(tui.LogClose, MsgPrefix+MsgExportFailed+err.Error())
		return err
	}

	if err := h.writeOutput(sql); err != nil {
		h.logFn(tui.LogClose, MsgPrefix+MsgWriteFailed+err.Error())
		return err
	}

	h.logFn(tui.LogClose, MsgPrefix+MsgExportSuccess+h.outputPath)
	return nil
}

func (h *Handler) writeOutput(sql string) error {
	dir := filepath.Dir(h.outputPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Err(err.Error())
		}
	}
	if err := os.WriteFile(h.outputPath, []byte(sql), 0o644); err != nil {
		return fmt.Err(err.Error())
	}
	return nil
}
