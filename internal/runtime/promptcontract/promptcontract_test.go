package promptcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestComposeArtifactOnlyPromptKeepsSharedOrderAcrossProviders(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	packDir := filepath.Join(workspaceDir, "skills", "prompt-packs")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("mkdir prompt pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "findings.md"), []byte("Prompt pack line A\nPrompt pack line B\n"), 0o644); err != nil {
		t.Fatalf("write prompt pack: %v", err)
	}

	task := acpruntime.Task{
		StepID:            "init.step3.findings",
		Workspace:         workspaceDir,
		WriteRoot:         filepath.Join(workspaceDir, "reports", "taskruns", "run-1", "validator"),
		DraftFinalRoot:    filepath.Join(workspaceDir, "reports", "taskruns", "run-1", "staging", "final"),
		ExpectedArtifacts: []string{"validator-verdict.json"},
	}

	claudePrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
	qwenPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)

	claudeTail := strings.TrimPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\n")
	qwenTail := strings.TrimPrefix(qwenPrompt, "You are ACP runtime provider \"qwen-code\".\n\n")
	if claudeTail != qwenTail {
		t.Fatalf("expected providers to share identical enforced prompt body\nclaude:\n%s\n\nqwen:\n%s", claudeTail, qwenTail)
	}

	expectedOrder := []string{
		"Artifact-only contract:",
		"Write ONLY inside write_root.",
		"Write validator-verdict.json in write_root.",
		"WORKSPACE PROMPT PACK CONTENT LAYER:",
		"Completion rule:",
	}
	lastIndex := -1
	for _, token := range expectedOrder {
		index := strings.Index(claudePrompt, token)
		if index < 0 {
			t.Fatalf("expected prompt to contain %q, got:\n%s", token, claudePrompt)
		}
		if index <= lastIndex {
			t.Fatalf("expected prompt token %q to appear after previous token; prompt:\n%s", token, claudePrompt)
		}
		lastIndex = index
	}
}
