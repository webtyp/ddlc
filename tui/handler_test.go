package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"webtyp.com/fmt"
)

const (
	msgExporting     = "Exporting DDL schema..."
	msgExportSuccess = "Export successful. Written to: "
	msgExportFailed  = "Export failed: "
)

func collectLogs(t *testing.T, h *Handler) *[]string {
	t.Helper()
	var logs []string
	h.SetLog(func(messages ...any) {
		var strMsgs []string
		for _, m := range messages {
			strMsgs = append(strMsgs, fmt.Sprint(m))
		}
		logs = append(logs, strings.Join(strMsgs, " "))
	})
	return &logs
}

func TestHandler_WithoutExportFunc(t *testing.T) {
	h := New()
	logs := collectLogs(t, h)

	err := h.ExecuteErr()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErr := "ddlc handler: export function not configured"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}

	found := false
	for _, log := range *logs {
		if strings.Contains(log, expectedErr) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error message in logs, logs were: %v", *logs)
	}
}

func TestHandler_WithExportFunc_Success_WritesFile(t *testing.T) {
	h := New()
	logs := collectLogs(t, h)

	outputPath := filepath.Join(t.TempDir(), "nested", "db.sql")
	h.SetOutputPath(outputPath)

	expectedSQL := "CREATE TABLE users (id INTEGER PRIMARY KEY);"
	h.SetExport(func() (string, error) {
		return expectedSQL, nil
	})

	if err := h.ExecuteErr(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	written, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
	if string(written) != expectedSQL {
		t.Errorf("expected written SQL %q, got %q", expectedSQL, string(written))
	}

	foundOpen, foundClose := false, false
	for _, log := range *logs {
		if strings.Contains(log, msgExporting) {
			foundOpen = true
		}
		if strings.Contains(log, msgExportSuccess) && strings.Contains(log, outputPath) {
			foundClose = true
		}
	}
	if !foundOpen {
		t.Error("expected LogOpen message in logs")
	}
	if !foundClose {
		t.Errorf("expected LogClose success message with output path in logs, logs were: %v", *logs)
	}
}

func TestHandler_DefaultOutputPath(t *testing.T) {
	h := New()
	if h.outputPath != DefaultOutputPath {
		t.Errorf("expected default output path %q, got %q", DefaultOutputPath, h.outputPath)
	}
}

// TestHandler_ExecuteErr_NothingToExport_ShowsFriendlyMessage_NoFileWritten is
// the ddlc/tui-side regression test for the reported bug: pressing the export
// button on a project with no models must NOT write a file, and must show an
// informative message instead of a generic failure. This exercises the
// generic ExportFunc contract (ErrNothingToExport), decoupled from any
// specific implementation like ormc — see errors.Is(err, ErrNothingToExport).
func TestHandler_ExecuteErr_NothingToExport_ShowsFriendlyMessage_NoFileWritten(t *testing.T) {
	h := New()
	outPath := filepath.Join(t.TempDir(), "db.sql")
	h.SetOutputPath(outPath)
	logs := collectLogs(t, h)
	h.SetExport(func() (string, error) { return "", ErrNothingToExport })

	err := h.ExecuteErr()
	if !errors.Is(err, ErrNothingToExport) {
		t.Fatalf("ExecuteErr() = %v, want ErrNothingToExport", err)
	}
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		t.Error("file should NOT have been written when there is nothing to export")
	}

	found := false
	for _, log := range *logs {
		if strings.Contains(log, MsgNoModels) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected friendly no-models message in logs, got: %v", *logs)
	}
}

func TestHandler_Label_ShortensPathRelativeToRootDir(t *testing.T) {
	h := New()
	root := "/home/user/project"
	h.SetOutputPath(filepath.Join(root, "config/schema.sql"))
	h.SetRootDir(root)

	label := h.Label()
	want := "Export SQL DDL → ./config/schema.sql"
	if label != want {
		t.Errorf("Label() = %q, want %q", label, want)
	}
	if strings.Contains(label, root) {
		t.Errorf("Label() = %q still contains the absolute root %q", label, root)
	}
}

func TestHandler_Label_WithoutRootDir_ShowsRawOutputPath(t *testing.T) {
	h := New()
	label := h.Label()
	want := "Export SQL DDL → " + DefaultOutputPath
	if label != want {
		t.Errorf("Label() = %q, want %q (no SetRootDir call => no shortening)", label, want)
	}
}

func TestHandler_WithExportFunc_Error(t *testing.T) {
	h := New()
	logs := collectLogs(t, h)

	expectedErr := fmt.Err("database connection lost")
	h.SetExport(func() (string, error) {
		return "", expectedErr
	})

	err := h.ExecuteErr()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected propagated error to be verbatim %v, got %v", expectedErr, err)
	}

	foundErr := false
	for _, log := range *logs {
		if strings.Contains(log, msgExportFailed) && strings.Contains(log, expectedErr.Error()) {
			foundErr = true
		}
	}
	if !foundErr {
		t.Errorf("expected error message in logs, logs were: %v", *logs)
	}
}

func TestHandler_Execute_Structural(t *testing.T) {
	h := New()
	h.SetOutputPath(filepath.Join(t.TempDir(), "db.sql"))
	called := false
	h.SetExport(func() (string, error) {
		called = true
		return "", nil
	})
	h.Execute()
	if !called {
		t.Error("expected export function to be called by Execute()")
	}
}
