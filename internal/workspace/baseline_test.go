package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureBaselineBundleCreatesMissingArtifacts(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}

	required := []string{
		"skills/subagents.yaml",
		"skills/prompt-packs/collect-context.md",
		"skills/service-inventory/prompts/system.md",
		"skills/service-inventory/prompts/task.md",
		"skills/service-inventory/templates/adr.md",
		"skills/service-inventory/templates/rfc.md",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(ws.Path, rel)); err != nil {
			t.Fatalf("expected %s to be seeded: %v", rel, err)
		}
	}
}

func TestEnsureBaselineBundleDoesNotOverwriteExistingFiles(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	customPrompt := "custom prompt content\n"
	if err := ws.WriteFile("skills/prompt-packs/collect-context.md", []byte(customPrompt)); err != nil {
		t.Fatalf("write custom prompt pack: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "skills/prompt-packs/collect-context.md"))
	if err != nil {
		t.Fatalf("read prompt pack: %v", err)
	}
	if string(content) != customPrompt {
		t.Fatalf("expected custom prompt pack to stay unchanged, got %q", string(content))
	}
}

func writeBaselineWorkspace(t *testing.T) Root {
	t.Helper()

	root := t.TempDir()
	manifest := strings.TrimSpace(`
version: 1
repos:
  - name: sample
    path: /tmp/sample
`) + "\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}
