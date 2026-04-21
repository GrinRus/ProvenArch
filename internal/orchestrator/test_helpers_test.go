package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func createWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repoA := filepath.Join(root, "repos", "payments-service")
	repoB := filepath.Join(root, "repos", "users-service")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("create payments repo: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("create users repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoA, "README.md"), []byte("# payments-service\n"), 0o644); err != nil {
		t.Fatalf("write payments readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoB, "README.md"), []byte("# users-service\n"), 0o644); err != nil {
		t.Fatalf("write users readme: %v", err)
	}
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoA + `
  - name: users-service
    path: ` + repoB + `
`
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}
