package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateReportsLayoutReadinessWarnings(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{})
	if !hasDiagnosticCode(report.Warnings, "workspace.layout.dir.missing") {
		t.Fatalf("expected layout readiness warning, got %+v", report.Warnings)
	}
}

func TestValidateReportsLayoutNotDirectoryAsError(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "reports", "as-is"), 0o755); err != nil {
		t.Fatalf("create reports/as-is: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "reports", "as-is", "services"), []byte("not-a-dir"), 0o644); err != nil {
		t.Fatalf("write conflicting services file: %v", err)
	}

	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	report := ws.Validate(context.Background(), ValidateOptions{})
	if report.OK {
		t.Fatalf("expected report.OK=false due to invalid layout")
	}
	if !hasDiagnosticCode(report.Errors, "workspace.layout.dir.not_dir") {
		t.Fatalf("expected layout not_dir error, got %+v", report.Errors)
	}
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
