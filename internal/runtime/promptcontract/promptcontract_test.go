package promptcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
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
		"VALIDATOR FIRST-ACTION ARTIFACT:",
		"FIRST VALIDATOR VERDICT COMMAND:",
		"cat > ",
		"Artifact-only contract:",
		"Write ONLY inside write_root.",
		"Every required artifact write/check MUST use the exact absolute write_root or draft_final_root paths above.",
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
	if got := strings.Count(claudePrompt, "FIRST VALIDATOR VERDICT COMMAND:"); got != 1 {
		t.Fatalf("expected validator first-action command heading exactly once, got %d:\n%s", got, claudePrompt)
	}
	if got := strings.Count(claudePrompt, "ACP_VALIDATOR_VERDICT_JSON"); got != 2 {
		t.Fatalf("expected one validator verdict heredoc in normal prompt, got delimiter count %d:\n%s", got, claudePrompt)
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
		"COLLECT MANIFEST CONTRACT CHECKLIST:",
		"Early pair-write requirement",
		"Absolute collect targets for the early pair-write",
		"semantic.coverage MUST use observed/missing/notes",
		"semantic.questions[*] MUST use id + text",
		"semantic.findings[*] MUST include id, severity, title, and provenance",
		"Do NOT add top-level step_contract, compatibility",
		"semantic provenance.evidence[*] objects MUST carry repo/path",
		"The task-specific collect manifest JSON skeleton above is normative",
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
		`"repo": "payments-service"`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected collect prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if strings.Contains(prompt, artifactquality.CollectManifestCanonicalExample()) {
		t.Fatalf("collect prompt should not include the generic collect manifest example:\n%s", prompt)
	}
}

func TestComposeArtifactOnlyPromptKeepsRefreshCollectFirstActionTaskSpecific(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/root-files",
		WriteRoot:    "/tmp/write-root",
		RepoScopes:   []string{"bank"},
		PathScopes:   []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
		ShardID:      "bank-root-files",
		DomainID:     "bank",
		AgentRole:    "shard-analyst",
	}

	claudePrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
	qwenPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)
	codexPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)

	claudeTail := strings.TrimPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\n")
	qwenTail := strings.TrimPrefix(qwenPrompt, "You are ACP runtime provider \"qwen-code\".\n\n")
	codexTail := strings.TrimPrefix(codexPrompt, "You are ACP runtime provider \"codex-code\".\n\n")
	if claudeTail != qwenTail || claudeTail != codexTail {
		t.Fatalf("expected collect providers to share identical enforced prompt body\nclaude:\n%s\n\nqwen:\n%s\n\ncodex:\n%s", claudeTail, qwenTail, codexTail)
	}
	prompt := qwenPrompt
	for _, token := range []string{
		"COLLECT FIRST-ACTION ARTIFACT PAIR:",
		"FIRST COLLECT ARTIFACT PAIR COMMAND:",
		"Artifact-only contract:",
		"DOCS-FIRST FILESYSTEM CONTRACT:",
		`"step_id": "refresh.step1.collect"`,
		`"path": "README.md"`,
		`"questions": [`,
		`"missing": [`,
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected refresh collect prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if !strings.HasPrefix(prompt, "You are ACP runtime provider \"qwen-code\".\n\nCOLLECT FIRST-ACTION ARTIFACT PAIR:") {
		t.Fatalf("expected collect first-action section immediately after provider identity, got:\n%s", prompt)
	}
	if got := strings.Count(prompt, "FIRST COLLECT ARTIFACT PAIR COMMAND:"); got != 1 {
		t.Fatalf("expected first-action command heading exactly once, got %d:\n%s", got, prompt)
	}
	firstActionIndex := strings.Index(prompt, "FIRST COLLECT ARTIFACT PAIR COMMAND:")
	artifactContractIndex := strings.Index(prompt, "Artifact-only contract:")
	docFirstIndex := strings.Index(prompt, "DOCS-FIRST FILESYSTEM CONTRACT:")
	if firstActionIndex < 0 || artifactContractIndex < 0 || docFirstIndex < 0 {
		t.Fatalf("expected first-action, artifact contract, and doc-first sections:\n%s", prompt)
	}
	if !(firstActionIndex < artifactContractIndex && artifactContractIndex < docFirstIndex) {
		t.Fatalf("expected collect first-action command before broad artifact/doc-first contract:\n%s", prompt)
	}
}

func TestComposeCollectManifestRepairPromptIsManifestOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeRoot := filepath.Join(root, "write-root")
	repoRoot := filepath.Join(root, "payments-repo")
	if err := os.MkdirAll(filepath.Join(repoRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "overview.md"), []byte("# Payments\n"), 0o644); err != nil {
		t.Fatalf("write authored doc: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(writeRoot, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir nested authored docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "docs", "deep-dive.md"), []byte("# Payments Deep Dive\n"), 0o644); err != nil {
		t.Fatalf("write nested authored doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "src", "README.md"), []byte("# Payments source\n"), 0o644); err != nil {
		t.Fatalf("write evidence candidate: %v", err)
	}

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/payments",
		WriteRoot:        writeRoot,
		ReadContextRoots: []string{repoRoot},
		ShardID:          "payments",
		DomainID:         "payments",
		AgentRole:        "shard-analyst",
		RepoScopes:       []string{"payments-service"},
		PathScopes:       []string{"src"},
	}

	prompt := ComposeCollectManifestRepairPrompt(acpruntime.ProviderQwenCode, task, os.ErrNotExist)
	expectedTokens := []string{
		"collect manifest repair mode",
		"Run the preferred file write command below",
		"mkdir -p ",
		"cat > ",
		"<<'ACP_MANIFEST_JSON'",
		"do not inspect or patch it; overwrite it from the heredoc command",
		"Copy the heredoc JSON exactly during repair",
		"Write or replace only write_root/shard-pack-manifest.json.",
		"Exact allowed write target:",
		"Final action must be: write only write_root/shard-pack-manifest.json, then exit successfully.",
		"Existing authored documents in write_root are already encoded in the task-specific skeleton; do not rewrite them.",
		"Do not search the filesystem for schemas/*, docs/spec/*, examples, prior manifests",
		"sibling shards, raw logs, or reports/taskruns history",
		"Existing authored document files in write_root:",
		"docs/deep-dive.md",
		"overview.md",
		"Repository evidence candidates are already encoded in the skeleton",
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		`"path": "docs/deep-dive.md"`,
		`"path": "overview.md"`,
		`"path": "src/README.md"`,
		"COLLECT MANIFEST REPAIR INSTRUCTIONS:",
		"Execute the preferred heredoc write command",
		"Do not read, diff, or patch an existing invalid shard-pack-manifest.json; replace it.",
		"COLLECT MANIFEST REPAIR CHECKLIST:",
		`artifact_root must remain exactly "reports/taskruns/run-1/staging/shards/payments"`,
		"forbidden legacy aliases:",
		"complete repair artifact; write it exactly from the heredoc command",
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if strings.Contains(prompt, "Produce runtime-authored documents in write_root") {
		t.Fatalf("repair prompt must not ask the provider to rewrite authored docs:\n%s", prompt)
	}
	if strings.Contains(prompt, "Previous artifact contract failure") {
		t.Fatalf("repair prompt should not include validation-error cues that invite patching instead of heredoc overwrite:\n%s", prompt)
	}
	skeletonIndex := strings.Index(prompt, "TASK-SPECIFIC MANIFEST JSON SKELETON:")
	contractIndex := strings.Index(prompt, "Artifact-only repair contract:")
	instructionIndex := strings.Index(prompt, "COLLECT MANIFEST REPAIR INSTRUCTIONS:")
	checklistIndex := strings.Index(prompt, "COLLECT MANIFEST REPAIR CHECKLIST:")
	if skeletonIndex < 0 || contractIndex < 0 || instructionIndex < 0 || checklistIndex < 0 {
		t.Fatalf("expected repair prompt to contain skeleton, instructions, and canonical sections:\n%s", prompt)
	}
	if !(skeletonIndex < contractIndex && contractIndex < instructionIndex && instructionIndex < checklistIndex) {
		t.Fatalf("expected compact repair prompt to put exact skeleton before contract, repair instructions, and canonical reference:\n%s", prompt)
	}
	if strings.Count(prompt, "TASK-SPECIFIC MANIFEST JSON SKELETON:") != 1 {
		t.Fatalf("repair prompt should include exactly one task-specific JSON skeleton section:\n%s", prompt)
	}
	if strings.Count(prompt, "ACP_MANIFEST_JSON") != 2 {
		t.Fatalf("repair prompt should include a single heredoc command around the skeleton:\n%s", prompt)
	}
	if strings.Contains(prompt, artifactquality.CollectManifestCanonicalExample()) {
		t.Fatalf("repair prompt should not include a second generic canonical JSON example after the task skeleton:\n%s", prompt)
	}
}

func TestComposeCollectManifestRepairPromptPrefersUsefulRootEvidence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeRoot := filepath.Join(root, "write-root")
	repoRoot := filepath.Join(root, "bank-repo")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	for _, name := range []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"} {
		if err := os.WriteFile(filepath.Join(repoRoot, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatalf("write repo file %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "root-overview.md"), []byte("# Root Overview\n"), 0o644); err != nil {
		t.Fatalf("write authored doc: %v", err)
	}

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "refresh.step1.collect",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/bank-root-files",
		WriteRoot:        writeRoot,
		ReadContextRoots: []string{repoRoot},
		ShardID:          "bank-root-files",
		DomainID:         "bank",
		AgentRole:        "shard-analyst",
		RepoScopes:       []string{"bank"},
		PathScopes:       []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
	}

	prompt := ComposeCollectManifestRepairPrompt(acpruntime.ProviderQwenCode, task, os.ErrNotExist)
	if !strings.Contains(prompt, `"path": "README.md"`) {
		t.Fatalf("expected repair prompt skeleton to prefer README.md citation, got:\n%s", prompt)
	}
	if strings.Contains(prompt, `"path": ".gitignore"`) {
		t.Fatalf("repair prompt skeleton must not choose .gitignore as primary root evidence, got:\n%s", prompt)
	}
}

func TestComposeCollectArtifactPairRepairPromptWritesExactPairFirst(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step1.collect",
		ArtifactRoot:      "reports/taskruns/run-1/staging/shards/payments",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments",
		ShardID:           "payments",
		DomainID:          "payments",
		AgentRole:         "shard-analyst",
		RepoScopes:        []string{"payments-service"},
		PathScopes:        []string{"."},
		ExpectedArtifacts: []string{"shard-pack-manifest.json"},
	}

	prompt := ComposeCollectArtifactPairRepairPrompt(acpruntime.ProviderCodexCode, task, os.ErrNotExist)
	expectedTokens := []string{
		"collect artifact pair focused recovery mode",
		"Run the exact shell command below as your next command. Do not inspect repository files first.",
		`Write exactly two files now: "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md" and "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json".`,
		"COLLECT PAIR WRITE COMMAND:",
		"cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md' <<'ACP_COLLECT_DOC'",
		"cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json' <<'ACP_MANIFEST_JSON'",
		`"path": "payments-overview.md"`,
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
		"exact authored document target = \"/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md\"",
		"exact manifest target = \"/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json\"",
		"Do not infer schema from prior reports/taskruns artifacts or raw logs.",
		"Previous collect artifact validation failure",
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected collect pair repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	for _, forbidden := range []string{
		"collect manifest repair mode",
		"Existing authored documents in write_root",
		"Do not rewrite existing authored markdown documents",
		"read, diff, or patch an existing invalid shard-pack-manifest.json",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("collect pair repair prompt must not use manifest-only repair wording %q:\n%s", forbidden, prompt)
		}
	}
	if strings.Count(prompt, "ACP_COLLECT_DOC") != 2 || strings.Count(prompt, "ACP_MANIFEST_JSON") != 2 {
		t.Fatalf("expected one collect doc heredoc and one manifest heredoc:\n%s", prompt)
	}
}

func TestComposeValidatorVerdictRepairPromptIsVerdictOnly(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step3.findings",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/validator",
		ReadContextRoots:  []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
		ExpectedArtifacts: []string{"validator-verdict.json"},
	}

	prompt := ComposeValidatorVerdictRepairPrompt(acpruntime.ProviderCodexCode, task, os.ErrNotExist)
	expectedTokens := []string{
		"validator verdict focused recovery mode",
		"Immediate validator verdict repair action:",
		"Run the exact shell command below as your next command",
		"Write exactly one file now:",
		"/tmp/workspace/reports/taskruns/run-1/validator/validator-verdict.json",
		"FIRST VALIDATOR VERDICT COMMAND:",
		"<<'ACP_VALIDATOR_VERDICT_JSON'",
		`"issues": []`,
		"VALIDATOR VERDICT REPAIR INSTRUCTIONS:",
		"The heredoc JSON is the complete first repair artifact",
		"issues[] items must use only: code, severity, message, path, document_id, citation_id",
		"Legacy issue fields are forbidden inside issues[]: id, title, description, rule_id, related_paths, related_ids, provenance",
		`"code": "staged_index_missing"`,
		`"severity": "error"`,
		"Previous validator artifact validation failure",
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected validator repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if strings.Contains(prompt, "shard-pack-manifest.json") && !strings.Contains(prompt, "Do not write shard-pack-manifest.json") {
		t.Fatalf("validator repair prompt must only mention shard manifest as forbidden:\n%s", prompt)
	}
	if strings.Count(prompt, "ACP_VALIDATOR_VERDICT_JSON") != 2 {
		t.Fatalf("expected one validator verdict heredoc:\n%s", prompt)
	}
	if strings.Contains(prompt, "VALIDATOR VERDICT JSON SKELETON:") {
		t.Fatalf("validator repair prompt must not duplicate the heredoc skeleton in a later section:\n%s", prompt)
	}
}

func TestComposeDraftArtifactRepairPromptNamesExactTargets(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step2.asis_docs",
		StepContract:      "as_is",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/asis",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ExpectedArtifacts: []string{"asis-draft-manifest.json", "overview.md", "summary.md", "architect-summary.md"},
	}

	prompt := ComposeDraftArtifactRepairPrompt(acpruntime.ProviderQwenCode, task, os.ErrNotExist)
	expectedTokens := []string{
		"draft artifact focused recovery mode",
		"Immediate draft artifact repair action:",
		"/tmp/workspace/reports/taskruns/run-1/asis/asis-draft-manifest.json",
		"/tmp/workspace/reports/taskruns/run-1/staging/final",
		"Do not begin with broad analysis. Run the preferred file write command below",
		"overwrite them from the heredoc artifacts",
		"RUNTIME DRAFT MANIFEST JSON SKELETON:",
		"cat > '/tmp/workspace/reports/taskruns/run-1/asis/asis-draft-manifest.json' <<'ACP_DRAFT_MANIFEST_JSON'",
		"cat > '/tmp/workspace/reports/taskruns/run-1/staging/final/overview.md' <<'ACP_DRAFT_FILE'",
		"cat > '/tmp/workspace/reports/taskruns/run-1/staging/final/summary.md' <<'ACP_DRAFT_FILE'",
		"cat > '/tmp/workspace/reports/taskruns/run-1/staging/final/architect-summary.md' <<'ACP_DRAFT_FILE'",
		"test -s '/tmp/workspace/reports/taskruns/run-1/asis/asis-draft-manifest.json'",
		"test -s '/tmp/workspace/reports/taskruns/run-1/staging/final/overview.md'",
		"test -s '/tmp/workspace/reports/taskruns/run-1/staging/final/summary.md'",
		"test -s '/tmp/workspace/reports/taskruns/run-1/staging/final/architect-summary.md'",
		`"step_contract": "as_is"`,
		`"path": "overview.md"`,
		"# Coverage Summary",
		"Every outputs[].path must be relative to draft_final_root",
		"Absolute target checks must use write_root/draft_final_root exactly",
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected draft repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if strings.Contains(prompt, artifactquality.AsIsDraftManifestCanonicalExample()) {
		t.Fatalf("draft repair prompt should stay compact and not include the full generic as-is canonical example")
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
		"VALIDATOR FIRST-ACTION ARTIFACT:",
		"FIRST VALIDATOR VERDICT COMMAND:",
		"cat > '/tmp/write-root/validator-verdict.json' <<'ACP_VALIDATOR_VERDICT_JSON'",
		"VALIDATOR VERDICT CANONICAL SHAPE:",
		"validator-verdict.json MUST include version=1, run_id, generated_at, verdict, and checked_paths.",
		"issues[] items MUST use exactly the canonical validator issue shape",
		"Do NOT put legacy finding-shaped fields inside issues[]",
		"owner-only residual evidence gaps may still return verdict=PASS when no technical validator issues remain.",
		`"generated_at": "2026-04-16T12:00:02Z"`,
		`"code": "staged_index_missing"`,
		`"title": "Owner mapping remains unresolved"`,
		`"repo": "payments-service"`,
		`"path": "README.md"`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected findings prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	firstActionIndex := strings.Index(prompt, "FIRST VALIDATOR VERDICT COMMAND:")
	artifactContractIndex := strings.Index(prompt, "Artifact-only contract:")
	if firstActionIndex < 0 || artifactContractIndex < 0 || firstActionIndex > artifactContractIndex {
		t.Fatalf("expected validator first-action command before broad artifact-only contract:\n%s", prompt)
	}
	if got := strings.Count(prompt, "FIRST VALIDATOR VERDICT COMMAND:"); got != 1 {
		t.Fatalf("expected one validator first-action command heading, got %d:\n%s", got, prompt)
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

func TestComposeArtifactOnlyPromptAddsProposalsDraftCanonicalSection(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		StepID:            "init.step4.proposals",
		WriteRoot:         "/tmp/write-root",
		DraftFinalRoot:    "/tmp/draft-root",
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
	}

	claudePrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
	qwenPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)
	codexPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)

	claudeTail := strings.TrimPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\n")
	qwenTail := strings.TrimPrefix(qwenPrompt, "You are ACP runtime provider \"qwen-code\".\n\n")
	codexTail := strings.TrimPrefix(codexPrompt, "You are ACP runtime provider \"codex-code\".\n\n")
	if claudeTail != qwenTail || claudeTail != codexTail {
		t.Fatalf("expected proposals enforced prompt body to be provider-independent")
	}

	expectedTokens := []string{
		"PROPOSALS FIRST-ACTION DRAFT ARTIFACTS:",
		"FIRST PROPOSALS DRAFT COMMAND:",
		"cat > '/tmp/write-root/proposals-draft-manifest.json' <<'ACP_DRAFT_MANIFEST_JSON'",
		"cat > '/tmp/draft-root/proposal.md' <<'ACP_DRAFT_FILE'",
		"cat > '/tmp/draft-root/changelog.md' <<'ACP_DRAFT_FILE'",
		"PROPOSALS DRAFT MANIFEST CANONICAL SHAPE:",
		`proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="proposals", agent_role, optional summary, and outputs[].`,
		`outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.`,
		`pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps`,
		`"step_contract": "proposals"`,
		`"canonical_path": "proposals/proposal-baseline/proposal.md"`,
		`"canonical_path": "reports/changelog/run-1.md"`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(claudePrompt, token) {
			t.Fatalf("expected proposals prompt to contain %q, got:\n%s", token, claudePrompt)
		}
	}
	firstActionIndex := strings.Index(claudePrompt, "FIRST PROPOSALS DRAFT COMMAND:")
	artifactContractIndex := strings.Index(claudePrompt, "Artifact-only contract:")
	if firstActionIndex < 0 || artifactContractIndex < 0 || firstActionIndex > artifactContractIndex {
		t.Fatalf("expected proposals first-action command before broad artifact-only contract:\n%s", claudePrompt)
	}
	if got := strings.Count(claudePrompt, "FIRST PROPOSALS DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected one proposals first-action command heading, got %d:\n%s", got, claudePrompt)
	}
	if got := strings.Count(claudePrompt, "ACP_DRAFT_MANIFEST_JSON"); got != 2 {
		t.Fatalf("expected one proposals manifest heredoc in normal prompt, got delimiter count %d:\n%s", got, claudePrompt)
	}
	if got := strings.Count(claudePrompt, "ACP_DRAFT_FILE"); got != 4 {
		t.Fatalf("expected two proposals draft file heredocs in normal prompt, got delimiter count %d:\n%s", got, claudePrompt)
	}
}
