package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestValidateDocsImportsIndexAbsentIsSilent(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{})
	if hasDiagnosticCodePrefix(report.Warnings, "workspace.docs.imports_index.") {
		t.Fatalf("expected missing imports index to be silent, got %+v", report.Warnings)
	}
}

func TestValidateDocsImportsIndexValidEntriesPassWithoutWarnings(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.WriteFile("docs/imports/architecture-notes.md", []byte("Architecture notes.")); err != nil {
		t.Fatalf("write import: %v", err)
	}
	if err := ws.WriteFile("docs/imports/index.yaml", []byte("- id: architecture-notes\n  path: docs/imports/architecture-notes.md\n  source: confluence\n  checksum: sha256:abc\n  imported_at: 2026-05-04T00:00:00Z\n  source_updated_at: 2026-05-03T00:00:00Z\n  status: active\n")); err != nil {
		t.Fatalf("write imports index: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{})
	if hasDiagnosticCodePrefix(report.Warnings, "workspace.docs.imports_index.") {
		t.Fatalf("expected valid imports index without warnings, got %+v", report.Warnings)
	}
}

func TestValidateDocsImportsIndexMalformedIsWarningOnly(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.WriteFile("docs/imports/index.yaml", []byte("- id: broken\n  path: [\n")); err != nil {
		t.Fatalf("write malformed imports index: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{})
	if !report.OK {
		t.Fatalf("expected malformed imports index to be warning-only, got errors %+v", report.Errors)
	}
	if !hasDiagnosticCode(report.Warnings, "workspace.docs.imports_index.malformed") {
		t.Fatalf("expected malformed imports index warning, got %+v", report.Warnings)
	}
}

func TestValidateDocsImportsIndexSemanticIssuesAreWarningsOnly(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.WriteFile("docs/imports/known.md", []byte("Known import.")); err != nil {
		t.Fatalf("write import: %v", err)
	}
	if err := ws.WriteFile("docs/imports/index.yaml", []byte(strings.Join([]string{
		"- id: duplicate",
		"  path: docs/imports/known.md",
		"- id: duplicate",
		"  path: docs/imports/missing.md",
		"- id: outside",
		"  path: README.md",
		"- id: absolute",
		"  path: /tmp/absolute.md",
		"- id: traversal",
		"  path: ../outside.md",
		"- id: directory",
		"  path: docs/imports",
		"",
	}, "\n"))); err != nil {
		t.Fatalf("write semantic imports index: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{})
	if !report.OK {
		t.Fatalf("expected semantic imports index issues to be warning-only, got errors %+v", report.Errors)
	}
	for _, code := range []string{
		"workspace.docs.imports_index.duplicate_id",
		"workspace.docs.imports_index.path_missing",
		"workspace.docs.imports_index.path_outside_imports",
		"workspace.docs.imports_index.path_absolute",
		"workspace.docs.imports_index.path_invalid",
		"workspace.docs.imports_index.path_not_file",
	} {
		if !hasDiagnosticCode(report.Warnings, code) {
			t.Fatalf("expected warning %s, got %+v", code, report.Warnings)
		}
	}
}

func TestValidateDocsImportsIndexUsesConfiguredImportsPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
docs:
  imports_path: ./external/imports
`)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.WriteFile("external/imports/custom.md", []byte("Custom import.")); err != nil {
		t.Fatalf("write custom import: %v", err)
	}
	if err := ws.WriteFile("external/imports/index.yaml", []byte("- id: custom\n  path: external/imports/custom.md\n")); err != nil {
		t.Fatalf("write custom imports index: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{})
	if hasDiagnosticCodePrefix(report.Warnings, "workspace.docs.imports_index.") {
		t.Fatalf("expected configured imports index without warnings, got %+v", report.Warnings)
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

func hasDiagnosticCodePrefix(diagnostics []Diagnostic, prefix string) bool {
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Code, prefix) {
			return true
		}
	}
	return false
}
