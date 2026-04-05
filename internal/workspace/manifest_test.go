package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsManifestWithDuplicateRepoNames(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: duplicate
    path: /tmp/one
  - name: duplicate
    git_url: https://example.com/repo.git
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), `duplicate repo.name "duplicate"`) {
		t.Fatalf("expected duplicate repo error, got %v", err)
	}
}

func TestOpenRejectsManifestMissingPathAndGitURL(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: invalid
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "must contain exactly one of path or git_url") {
		t.Fatalf("expected path/git_url invariant error, got %v", err)
	}
}

func TestResolveRejectsAbsoluteAndTraversal(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	if _, err := ws.Resolve("/etc/passwd"); err != ErrPathAbsolute {
		t.Fatalf("expected ErrPathAbsolute, got %v", err)
	}
	if _, err := ws.Resolve("../outside"); err != ErrPathTraversal {
		t.Fatalf("expected ErrPathTraversal, got %v", err)
	}
}

func TestEnsureLayoutCreatesRequiredDirectories(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	for _, rel := range []string{
		"model/entities",
		"reports/as-is/services",
		"reports/changelog",
		"charter/cards/domains",
	} {
		abs := filepath.Join(root, rel)
		info, err := os.Stat(abs)
		if err != nil {
			t.Fatalf("stat %q: %v", rel, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", rel)
		}
	}
}

func writeValidWorkspaceRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
`)
	return root
}

func writeManifestFile(t *testing.T, root string, content string) {
	t.Helper()
	manifestPath := filepath.Join(root, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}
