package workspace

import (
	"os"
	"path/filepath"
	"sort"
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

func TestEnsureBaselineSupportBundleDoesNotSeedCanonicalSubagentsOutput(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineSupportBundle(); err != nil {
		t.Fatalf("ensure baseline support bundle: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ws.Path, "skills/subagents.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected support bundle to avoid seeding canonical subagents output, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "skills/prompt-packs/collect-context.md")); err != nil {
		t.Fatalf("expected support bundle to keep prompt packs available: %v", err)
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

func TestEnsureBaselineBundleSeedsStructuredPromptDefaults(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}

	requiredSections := []string{
		"## Goal",
		"## Inputs",
		"## Required Output Shape",
		"## Evidence Policy",
		"## Forbidden Behavior",
		"## Fallback When Unknown",
	}

	promptPaths := make([]string, 0, len(baselinePromptPacks)+len(baselineSkillIDs)*2)
	for pack := range baselinePromptPacks {
		promptPaths = append(promptPaths, filepath.Join("skills", "prompt-packs", pack+".md"))
	}
	for _, skill := range baselineSkillIDs {
		promptPaths = append(promptPaths,
			filepath.Join("skills", skill, "prompts", "system.md"),
			filepath.Join("skills", skill, "prompts", "task.md"),
		)
	}
	sort.Strings(promptPaths)

	for _, rel := range promptPaths {
		content, err := os.ReadFile(filepath.Join(ws.Path, rel))
		if err != nil {
			t.Fatalf("read prompt %s: %v", rel, err)
		}
		body := string(content)
		for _, section := range requiredSections {
			if !strings.Contains(body, section) {
				t.Fatalf("prompt %s missing required section %q", rel, section)
			}
		}
		if words := len(strings.Fields(body)); words < 70 {
			t.Fatalf("prompt %s is too short: %d words", rel, words)
		}
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
