package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRejectsRelativePath(t *testing.T) {
	t.Parallel()

	_, err := Open("relative/workspace")
	if err != ErrWorkspaceAbsolute {
		t.Fatalf("expected ErrWorkspaceAbsolute, got %v", err)
	}
}

func TestOpenRejectsManifestSymlinkOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "workspace.yaml")
	if err := os.WriteFile(outside, []byte("version: 1\nrepos:\n  - name: sample\n    path: /tmp/sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ManifestFileName)); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); err == nil {
		t.Fatal("expected manifest symlink escape to fail")
	}
}

func TestOpenRejectsMissingManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := Open(root)
	if err != ErrManifestMissing {
		t.Fatalf("expected ErrManifestMissing, got %v", err)
	}
}

func TestOpenAcceptsWorkspaceRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manifestPath := filepath.Join(root, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte("version: 1\nrepos:\n  - name: sample\n    path: /tmp/sample\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	got, err := Open(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != root {
		t.Fatalf("expected path %q, got %q", root, got.Path)
	}
	if got.ManifestPath != manifestPath {
		t.Fatalf("expected manifest %q, got %q", manifestPath, got.ManifestPath)
	}
	if got.Manifest.Version != 1 {
		t.Fatalf("expected manifest version 1, got %d", got.Manifest.Version)
	}
	if len(got.Manifest.Repos) != 1 || got.Manifest.Repos[0].Name != "sample" {
		t.Fatalf("expected sample repo in parsed manifest, got %+v", got.Manifest.Repos)
	}
	if got.Manifest.Docs.ImportsPath != "./docs/imports" {
		t.Fatalf("expected docs.imports_path default, got %q", got.Manifest.Docs.ImportsPath)
	}
}
