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
	codexPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)

	claudeTail := strings.TrimPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\n")
	qwenTail := strings.TrimPrefix(qwenPrompt, "You are ACP runtime provider \"qwen-code\".\n\n")
	codexTail := strings.TrimPrefix(codexPrompt, "You are ACP runtime provider \"codex-code\".\n\n")
	if claudeTail != qwenTail {
		t.Fatalf("expected providers to share identical enforced prompt body\nclaude:\n%s\n\nqwen:\n%s", claudeTail, qwenTail)
	}
	if claudeTail != codexTail {
		t.Fatalf("expected providers to share identical enforced prompt body\nclaude:\n%s\n\ncodex:\n%s", claudeTail, codexTail)
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

func TestComposeArtifactOnlyPromptAddsCollectLegacyHygieneSection(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		StepID:         "init.step1.collect",
		ArtifactRoot:   "reports/taskruns/run-1/staging/shards/payments",
		WriteRoot:      "/tmp/write-root",
		DraftFinalRoot: "/tmp/draft-root",
		RepoScopes:     []string{"payments-service"},
		PathScopes:     []string{"src"},
	}

	prompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
	expectedTokens := []string{
		"COLLECT MANIFEST CANONICAL SHAPE:",
		"semantic.coverage MUST use observed/missing/notes",
		"Do NOT add top-level step_contract, compatibility",
		"semantic provenance.evidence[*] objects MUST carry repo/path",
		"Canonical fragment below is normative",
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
		`"repo": "payments-service"`,
		`"description": "Repository evidence names the payments service but does not identify an owning team."`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected collect prompt to contain %q, got:\n%s", token, prompt)
		}
	}
}

func TestComposeArtifactOnlyPromptAddsValidatorVerdictCanonicalSection(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		StepID:            "init.step3.findings",
		WriteRoot:         "/tmp/write-root",
		DraftFinalRoot:    "/tmp/draft-root",
		ExpectedArtifacts: []string{"validator-verdict.json"},
	}

	prompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)
	expectedTokens := []string{
		"VALIDATOR VERDICT CANONICAL SHAPE:",
		"validator-verdict.json MUST include version=1, run_id, generated_at, verdict, and checked_paths.",
		"owner-only residual evidence gaps may still return verdict=PASS when no technical validator issues remain.",
		`"generated_at": "2026-04-16T12:00:02Z"`,
		`"title": "Owner mapping remains unresolved"`,
		`"repo": "payments-service"`,
		`"path": "README.md"`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected findings prompt to contain %q, got:\n%s", token, prompt)
		}
	}
}

func TestComposeArtifactOnlyPromptAddsAsIsDraftCanonicalSection(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		StepID:            "init.step2.asis_docs",
		WriteRoot:         "/tmp/write-root",
		DraftFinalRoot:    "/tmp/draft-root",
		ExpectedArtifacts: []string{"asis-draft-manifest.json"},
	}

	prompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)
	expectedTokens := []string{
		"AS-IS DRAFT MANIFEST CANONICAL SHAPE:",
		`asis-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="as_is", agent_role, and outputs[].`,
		`overview.md -> reports/as-is/overview.md`,
		`"step_contract": "as_is"`,
		`"canonical_path": "reports/as-is/payments/overview.md"`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected as-is prompt to contain %q, got:\n%s", token, prompt)
		}
	}
}
