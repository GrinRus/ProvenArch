package promptcontract

import (
	"errors"
	"fmt"
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
		"Evidence-write pair requirement",
		"The first collect filesystem work unit may contain only two mechanically simple commands",
		"Absolute collect targets for the evidence-backed pair",
		"semantic.coverage MUST use observed/missing/notes",
		"semantic.questions[*] MUST use id + text",
		"semantic.findings[*] MUST include id, severity, title, and provenance",
		"Do NOT add top-level step_contract, compatibility",
		"semantic provenance.evidence[*] objects MUST carry repo/path",
		"The task-specific collect manifest JSON skeleton above is normative",
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
		`"repo": "payments-service"`,
		"not be seed-only, scaffold-only, or copied unchanged from the skeleton",
		`must not describe itself as an initial/temporary artifact, interrupted evidence read, or content that "will be repaired"`,
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
		"COLLECT EVIDENCE-FIRST ARTIFACT PAIR:",
		"FIRST COLLECT BOUNDED WRITE ACTION:",
		"COLLECT FINAL WRITE REQUIREMENT:",
		"COLLECT MANIFEST TASK SKELETON:",
		"SKELETON USE:",
		"Artifact-only contract:",
		"DOCS-FIRST FILESYSTEM CONTRACT:",
		`"step_id": "refresh.step1.collect"`,
		`"path": "README.md"`,
		`"questions": [`,
		`"missing": [`,
		"next filesystem work unit may contain only two mechanically simple commands: one bounded evidence read/list, then one direct literal write of both exact targets",
		"inspect at most 8 representative entrypoint/build/config/source files",
		"Read at most the first 6000 bytes from any file.",
		"Do not emit analysis-only prose",
		"Before both targets exist, do not use Ruby, Node, Python, Perl, awk, jq, generated source-code strings",
		"direct shell heredoc/printf/tee literal content",
		"start by writing an evidence-backed artifact pair; do not write a seed-only pair",
		"Normal collect must not depend on collect_pair_repair as the expected path to success.",
		"Copying this skeleton unchanged is invalid",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected refresh collect prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	for _, forbidden := range []string{
		"FIRST COLLECT ARTIFACT PAIR COMMAND:",
		"<<'ACP_COLLECT_DOC'",
		"<<'ACP_MANIFEST_JSON'",
		"ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT",
		"bootstrap-only",
		"unchanged bootstrap pair is an artifact_quality blocker",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("normal collect prompt must not contain bootstrap marker/wording %q:\n%s", forbidden, prompt)
		}
	}
	if !strings.HasPrefix(prompt, "You are ACP runtime provider \"qwen-code\".\n\nCOLLECT EVIDENCE-FIRST ARTIFACT PAIR:") {
		t.Fatalf("expected collect evidence-first section immediately after provider identity, got:\n%s", prompt)
	}
	if got := strings.Count(prompt, "COLLECT MANIFEST TASK SKELETON:"); got != 1 {
		t.Fatalf("expected collect task skeleton section exactly once, got %d:\n%s", got, prompt)
	}
	firstActionIndex := strings.Index(prompt, "FIRST COLLECT BOUNDED WRITE ACTION:")
	artifactContractIndex := strings.Index(prompt, "Artifact-only contract:")
	docFirstIndex := strings.Index(prompt, "DOCS-FIRST FILESYSTEM CONTRACT:")
	if firstActionIndex < 0 || artifactContractIndex < 0 || docFirstIndex < 0 {
		t.Fatalf("expected evidence-first, artifact contract, and doc-first sections:\n%s", prompt)
	}
	if !(firstActionIndex < artifactContractIndex && artifactContractIndex < docFirstIndex) {
		t.Fatalf("expected collect evidence-first section before broad artifact/doc-first contract:\n%s", prompt)
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
		"COLLECT MANIFEST EVIDENCE-FIRST REPAIR:",
		"Repair shard-pack-manifest.json from the existing authored documents and bounded repository evidence; do not start with a placeholder scaffold.",
		"Do not run a separate read-only preflight",
		"Do not answer with a plan, status note, or analysis-only message before the write.",
		"Forbidden analysis-only phrases before the write: I have enough evidence",
		"The first command below is a write-first provider-authored command contract",
		"No deterministic helper writes the manifest for you",
		"Read only the listed repository evidence candidates if authored docs need support",
		"Repository evidence in citations/provenance must be file-level",
		"directories or missing paths must become coverage gaps/questions, never citation paths",
		"JSON syntax-only checks such as jq empty or python3 -m json.tool are insufficient",
		"prove every citation/provenance repo path is an existing file",
		"Write exactly one file:",
		"Do not rewrite existing authored markdown documents.",
		"overwrite it after the evidence pass",
		"Do not add top-level claims, claim_map, validation, metadata, compatibility, schema",
		"claim IDs belong only in citations[].claim_ids",
		"Literal SHARD, <shard>, <claim>, TODO, REPLACE_ME",
		"FIRST COLLECT MANIFEST REPAIR COMMAND:",
		"Execute one bounded filesystem command as your next action.",
		"The command must read authored markdown already in write_root",
		"write the final shard-pack-manifest.json before it returns",
		"preflight-only completion is a failed no-op repair",
		"First command contract:",
		"read bounded authored markdown under write_root",
		"verify every manifest citation/provenance repo path with file-level checks such as test -f, rg --files, or portable find ... -type f -print",
		"write the final provider-authored manifest",
		"file-level evidence path checks after the write",
		"Exact manifest write target:",
		"Authored markdown inputs already present under write_root:",
		"Bounded repository evidence candidates:",
		"SKELETON USE:",
		"Use this JSON only as the task-specific schema/key/type guide, not as final content.",
		"Copying this skeleton unchanged is invalid",
		"SEMANTIC EXTRACTION REQUIREMENT:",
		"Evidence-rich authored documents require concrete semantic.entities beyond the repo plus shard wrapper.",
		"Evidence-rich authored documents require concrete semantic.edges beyond repo/shard contains relationships",
		"semantic.findings or semantic.questions beyond a generic owner-mapping gap",
		"A manifest with many citations but only repo/shard entities, only contains edges, and only Owner mapping not confirmed is invalid scaffold-only semantic output.",
		"Write or replace only write_root/shard-pack-manifest.json.",
		"not examined in this bounded pass",
		"not confirmed in scoped repository evidence",
		"Exact allowed write target:",
		"Final action must be: write only write_root/shard-pack-manifest.json, then exit successfully.",
		"Backend validation, not stdout claims, is the success surface.",
		"Existing authored documents in write_root must drive manifest repair; read them, do not rewrite them.",
		"Do not search the filesystem for schemas/*, docs/spec/*, examples, prior manifests",
		"sibling shards, raw logs, or reports/taskruns history",
		"Existing authored document files in write_root:",
		"docs/deep-dive.md",
		"overview.md",
		"Repository evidence candidates available for bounded repair:",
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		`"path": "docs/deep-dive.md"`,
		`"path": "overview.md"`,
		`"path": "src/README.md"`,
		`"entities": [`,
		`"edges": [`,
		`"findings": [`,
		"COLLECT MANIFEST REPAIR INSTRUCTIONS:",
		"Perform a bounded evidence pass over existing authored documents and listed repository evidence candidates inside the same first command that writes shard-pack-manifest.json.",
		"Do not read, diff, or patch an existing invalid shard-pack-manifest.json; replace it after the evidence pass.",
		"Do not stop after status prose such as \"I have enough evidence\"",
		"Do not cite directory paths in citations/provenance",
		"Syntax-valid JSON alone is not a valid repair",
		"Treat the embedded JSON as a schema guide only.",
		"Do not add top-level claims/claim_map/metadata/validation/compatibility",
		"do not preserve placeholder tokens such as SHARD, <shard>, <claim>, TODO, or REPLACE_ME",
		"Do not collapse semantic output to repo/shard wrappers plus owner mapping",
		"COLLECT MANIFEST REPAIR CHECKLIST:",
		`artifact_root must remain exactly "reports/taskruns/run-1/staging/shards/payments"`,
		"forbidden legacy aliases:",
		"copied scaffold semantic is invalid",
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
	for _, forbidden := range []string{
		"complete valid repair artifact",
		"complete repair artifact",
		"Do not make factual edits before the file validates",
		"Copy the heredoc JSON",
		"Execute the preferred heredoc write command",
		"write it from the heredoc command",
		`target.write_text(`,
		`"writes_manifest":true`,
		"read-only evidence preflight",
		"After the preflight",
		"writes_manifest",
		"ACP_COLLECT_MANIFEST_REPAIR_PY",
		"collect_manifest_repair_preflight",
		"preserve semantic.entities",
		"COLLECT MANIFEST REPAIR WRITE SHAPE:",
		`"<replace with authored doc objects from write_root>"`,
		`"<named systems/services/components/datastores/config surfaces>"`,
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("repair prompt must not frame the artifact as a content-free valid skeleton via %q:\n%s", forbidden, prompt)
		}
	}
	skeletonIndex := strings.Index(prompt, "TASK-SPECIFIC MANIFEST JSON SKELETON:")
	evidenceIndex := strings.Index(prompt, "COLLECT MANIFEST EVIDENCE-FIRST REPAIR:")
	commandIndex := strings.Index(prompt, "FIRST COLLECT MANIFEST REPAIR COMMAND:")
	skeletonUseIndex := strings.Index(prompt, "SKELETON USE:")
	semanticIndex := strings.Index(prompt, "SEMANTIC EXTRACTION REQUIREMENT:")
	contractIndex := strings.Index(prompt, "Artifact-only repair contract:")
	instructionIndex := strings.Index(prompt, "COLLECT MANIFEST REPAIR INSTRUCTIONS:")
	checklistIndex := strings.Index(prompt, "COLLECT MANIFEST REPAIR CHECKLIST:")
	if evidenceIndex < 0 || commandIndex < 0 || skeletonIndex < 0 || skeletonUseIndex < 0 || semanticIndex < 0 || contractIndex < 0 || instructionIndex < 0 || checklistIndex < 0 {
		t.Fatalf("expected repair prompt to contain skeleton, instructions, and canonical sections:\n%s", prompt)
	}
	if !(evidenceIndex < commandIndex && commandIndex < skeletonIndex && skeletonIndex < skeletonUseIndex && skeletonUseIndex < semanticIndex && semanticIndex < contractIndex && contractIndex < instructionIndex && instructionIndex < checklistIndex) {
		t.Fatalf("expected repair prompt to put evidence-first instructions before skeleton guide, contract, repair instructions, and canonical reference:\n%s", prompt)
	}
	if strings.Count(prompt, "TASK-SPECIFIC MANIFEST JSON SKELETON:") != 1 {
		t.Fatalf("repair prompt should include exactly one task-specific JSON skeleton section:\n%s", prompt)
	}
	if strings.Count(prompt, "COLLECT MANIFEST EVIDENCE-FIRST REPAIR:") != 1 {
		t.Fatalf("repair prompt should include exactly one evidence-first repair section:\n%s", prompt)
	}
	if strings.Count(prompt, "SEMANTIC EXTRACTION REQUIREMENT:") != 1 {
		t.Fatalf("repair prompt should include exactly one semantic extraction requirement section:\n%s", prompt)
	}
	if strings.Count(prompt, "FIRST COLLECT MANIFEST REPAIR COMMAND:") != 1 {
		t.Fatalf("repair prompt should include exactly one first-command section:\n%s", prompt)
	}
	if strings.Contains(prompt, artifactquality.CollectManifestCanonicalExample()) {
		t.Fatalf("repair prompt should not include a second generic canonical JSON example after the task skeleton:\n%s", prompt)
	}
}

func TestCollectManifestRepairGuidanceIsWriteFirst(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeRoot := filepath.Join(root, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	doc := strings.Join([]string{
		"# Payments Runtime Overview",
		"",
		"## Technology Stack",
		"- **Django API** serves payment workflow requests.",
		"- **Postgres** stores payment ledgers.",
		"- **Redis** backs async coordination.",
		"",
		"## Operational Gaps",
		"- Owner mapping and escalation still need confirmation.",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(writeRoot, "overview.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("write authored doc: %v", err)
	}
	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/payments",
		WriteRoot:    writeRoot,
		ShardID:      "payments",
		DomainID:     "payments",
		AgentRole:    "shard-analyst",
		RepoScopes:   []string{"payments-service"},
		PathScopes:   []string{"src"},
	}

	guidance := collectManifestRepairWriteFirstGuidance(task, []string{"overview.md"}, []string{"src/README.md"})
	for _, token := range []string{
		"First command contract:",
		"read bounded authored markdown under write_root",
		"verify every manifest citation/provenance repo path with file-level checks",
		"write the final provider-authored manifest",
		"run a local `test -s`/JSON parse check plus file-level evidence path checks after the write",
		filepath.Join(writeRoot, "shard-pack-manifest.json"),
		"overview.md",
		"src/README.md",
	} {
		if !strings.Contains(guidance, token) {
			t.Fatalf("expected repair guidance to contain %q, got:\n%s", token, guidance)
		}
	}
	for _, forbidden := range []string{
		"collect_manifest_repair_preflight",
		"writes_manifest",
		"read-only",
		"print(json.dumps",
	} {
		if strings.Contains(guidance, forbidden) {
			t.Fatalf("repair guidance must not preserve old preflight token %q:\n%s", forbidden, guidance)
		}
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
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

func TestCollectRepairEvidenceCandidatesSkipLargeAndGeneratedFiles(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for _, file := range []struct {
		path string
		body string
	}{
		{path: "README.md", body: "# Runtime\n"},
		{path: ".test_durations", body: strings.Repeat("slow-test\n", 20000)},
		{path: "pnpm-lock.yaml", body: "lockfileVersion: '9.0'\n"},
		{path: "uv.lock", body: strings.Repeat("package = []\n", 20000)},
		{path: "large-config.json", body: strings.Repeat("{\"entry\":true}\n", 12000)},
	} {
		if err := os.WriteFile(filepath.Join(repoRoot, file.path), []byte(file.body), 0o644); err != nil {
			t.Fatalf("write %s: %v", file.path, err)
		}
	}
	task := acpruntime.Task{
		ReadContextRoots: []string{repoRoot},
		PathScopes:       []string{"README.md", ".test_durations", "pnpm-lock.yaml", "uv.lock", "large-config.json"},
	}

	candidates := repairEvidenceCandidates(task)
	if got := strings.Join(candidates, ","); got != "README.md" {
		t.Fatalf("expected only useful bounded evidence candidate, got %q", got)
	}
}

func TestCollectRepairEvidenceCandidatesAreRankedAndCapped(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	names := []string{
		".cursorignore",
		".dockerignore",
		".editorconfig",
		".gitignore",
		".watchmanconfig",
		"README.md",
		"AGENTS.md",
		"CONTRIBUTING.md",
		"package.json",
		"pyproject.toml",
		"docker-compose.base.yml",
		"docker-compose.dev.yml",
		"Dockerfile",
		"Dockerfile.node",
		".env.example",
		"posthog.json",
		"pnpm-workspace.yaml",
		"tsconfig.json",
		"pytest.ini",
		"tach.toml",
		"turbo.json",
		"LICENSE",
		"Makefile",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(repoRoot, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	task := acpruntime.Task{
		ReadContextRoots: []string{repoRoot},
		PathScopes:       names,
	}

	candidates := repairEvidenceCandidates(task)
	if len(candidates) != 18 {
		t.Fatalf("expected capped candidate set of 18, got %d: %#v", len(candidates), candidates)
	}
	wantPrefix := []string{"AGENTS.md", "CONTRIBUTING.md", "README.md", "package.json", "pyproject.toml"}
	for i, want := range wantPrefix {
		if candidates[i] != want {
			t.Fatalf("expected ranked candidate %d to be %q, got %#v", i, want, candidates)
		}
	}
	for _, forbidden := range []string{".cursorignore", ".dockerignore", ".editorconfig", ".gitignore", ".watchmanconfig"} {
		for _, got := range candidates {
			if got == forbidden {
				t.Fatalf("expected dotfile %q to fall outside capped repair evidence, got %#v", forbidden, candidates)
			}
		}
	}
}

func TestCollectRepairEvidenceCandidatesKeepEachPathScope(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for _, file := range []string{
		"cli/README.md",
		"cli/Cargo.toml",
		"cli/CHANGELOG.md",
		"cli/src/api/client.rs",
		"cli/src/main.rs",
		"cli/tests/cli.rs",
		"common/README.md",
		"common/hogvm/README.md",
		"common/hogql_parser/pyproject.toml",
		"common/hogli/README.md",
	} {
		path := filepath.Join(repoRoot, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte(file+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}
	task := acpruntime.Task{
		ReadContextRoots: []string{repoRoot},
		PathScopes:       []string{"cli", "common"},
	}

	candidates := repairEvidenceCandidates(task)
	joined := strings.Join(candidates, "\n")
	for _, want := range []string{"cli/README.md", "cli/Cargo.toml", "common/README.md", "common/hogql_parser/pyproject.toml"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected multi-scope repair candidates to include %q, got:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "cli/package.json") || strings.Contains(joined, "common/hogvm/python/README.md") {
		t.Fatalf("repair candidates must only list existing files, got:\n%s", joined)
	}
}

func TestComposeCollectArtifactPairRepairPromptIsEvidenceFirstNoSeed(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Payments\n\nRuntime entrypoint.\n"), 0o644); err != nil {
		t.Fatalf("write evidence file: %v", err)
	}
	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step1.collect",
		ArtifactRoot:      "reports/taskruns/run-1/staging/shards/payments",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments",
		ReadContextRoots:  []string{repoRoot},
		ShardID:           "payments",
		DomainID:          "payments",
		AgentRole:         "shard-analyst",
		RepoScopes:        []string{"payments-service"},
		PathScopes:        []string{"README.md"},
		ExpectedArtifacts: []string{"shard-pack-manifest.json"},
	}

	prompt := ComposeCollectArtifactPairRepairPrompt(acpruntime.ProviderCodexCode, task, os.ErrNotExist)
	expectedTokens := []string{
		"collect artifact pair focused recovery mode",
		"COLLECT PAIR WRITE-FIRST EVIDENCE REPAIR:",
		"This repair is not a bootstrap/fallback writer",
		"Do not run a separate read-only preflight",
		"Do not answer with a plan, status note, or analysis-only message before the writes",
		"Forbidden analysis-only phrases before the writes: I have enough evidence",
		"I am now writing",
		"Your next action must be one bounded filesystem command",
		"at most 8 files and at most the first 6000 bytes from each file",
		"The first command must not contain a hard-coded required phrase list",
		"missing expected evidence",
		"Before writing, the only allowed evidence prechecks are structural",
		"at least one allowed evidence file yielded bytes after bounded prefix reads",
		"truncate to the first 6000 bytes or skip that candidate and continue",
		"do not abort the repair with errors such as `read file exceeds size limit`",
		"Do not add any other pre-write `raise SystemExit`/`exit 1` checks",
		"Keep the first command mechanically simple: no Python f-strings",
		"build JSON as dictionaries/lists, and write it with `json.dumps`",
		"If a planned claim is not present in the snippets, omit it or record a coverage gap",
		"The final markdown must be operator-facing architecture evidence, not a recovery/process log",
		"Write the markdown document first as concise evidence-backed content, then write the manifest that references it",
		`Write exactly two files in the first command: "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/root-overview.md" and "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json".`,
		"Do not delete existing files, run rm -f, use git rev-parse, rely on cwd discovery",
		"FIRST COLLECT PAIR REPAIR WRITE-FIRST COMMAND:",
		"Execute one filesystem command now",
		"final provider-authored evidence-backed content",
		"COLLECT PAIR REPAIR EVIDENCE LIMITS:",
		"Use only the listed task metadata and bounded repository evidence candidates below",
		"Do not read lockfiles, generated baselines, test duration indexes",
		"Do not verify claims with a provider-invented exact phrase checklist",
		"Use short observed snippets, package names, service names, config keys, and file paths from the bytes actually read",
		"read a bounded prefix or skip oversized candidates",
		"Do not abort before writing because your own generated script has no entities/edges/findings",
		"Every citation/provenance path must be a concrete existing repository file",
		"directories and missing paths are coverage gaps/questions, not evidence",
		"Do not add top-level claims, claim_map, validation, metadata, compatibility, schema",
		"claim IDs belong only in citations[].claim_ids",
		"Literal SHARD, <shard>, <claim>, TODO, REPLACE_ME",
		"JSON syntax-only checks such as jq empty or python3 -m json.tool are insufficient",
		"include file-level citation/provenance checks before a successful exit",
		"Use python3, not python; do not use GNU-only find -printf; do not assign to the zsh-reserved status variable.",
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		"RECOVERY ACCEPTANCE REQUIREMENT:",
		"Recovery Evidence Summary",
		"seed-only collect recovery fallback",
		"Additional provider enrichment should replace",
		"first bounded evidence read was attempted",
		"will be repaired with concrete",
		"Successful recovery output must not mention bounded read, bounded pass, guessed path, guessed file, guessed evidence",
		"not examined in this bounded pass",
		"not confirmed in scoped repository evidence",
		"FINAL SELF-CHECK COMMAND:",
		"! grep -E",
		"Successful recovery output must not contain ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT",
		"A noop, zero-output, unchanged skeleton, or partially-written repair is terminal",
		`"path": "root-overview.md"`,
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
		"exact authored document target = \"/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/root-overview.md\"",
		"exact manifest target = \"/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json\"",
		"Bounded repository evidence candidates:",
		"- README.md",
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
		"FIRST COLLECT PAIR REPAIR PREFLIGHT COMMAND:",
		"ACP_COLLECT_PAIR_REPAIR_PREFLIGHT_PY",
		"collect_pair_repair_preflight",
		"COLLECT PAIR WRITE COMMAND:",
		"cat > ",
		"ACP_COLLECT_DOC",
		"ACP_MANIFEST_JSON",
		"sed -n",
		"The command writes a marker-free seed recovery pair",
		"Exiting successfully immediately after the heredoc command is invalid",
		"POST-COMMAND ENRICHMENT REQUIREMENT:",
		"marker-bearing recovery bootstrap pair",
		"Evidence candidate used for the recovery manifest",
		"unchanged bootstrap pair is an artifact_quality blocker",
		"raise SystemExit('read file exceeds size limit')",
		`raise SystemExit("read file exceeds size limit")`,
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("collect pair repair prompt must not use manifest-only repair wording %q:\n%s", forbidden, prompt)
		}
	}
}

func TestComposeCollectArtifactPairRepairPromptTargetsExistingAuthoredMarkdown(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Payments\n\nRuntime entrypoint.\n"), 0o644); err != nil {
		t.Fatalf("write evidence file: %v", err)
	}
	writeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(writeRoot, "payments-overview.md"), []byte("# stale\n\n`missing-evidence.md`\n"), 0o644); err != nil {
		t.Fatalf("write authored doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "aaa-general.md"), []byte("# general\n\nREADME.md\n"), 0o644); err != nil {
		t.Fatalf("write secondary authored doc: %v", err)
	}
	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step1.collect",
		ArtifactRoot:      "reports/taskruns/run-1/staging/shards/payments",
		WriteRoot:         writeRoot,
		ReadContextRoots:  []string{repoRoot},
		ShardID:           "payments",
		DomainID:          "payments",
		AgentRole:         "shard-analyst",
		RepoScopes:        []string{"payments-service"},
		PathScopes:        []string{"README.md"},
		ExpectedArtifacts: []string{"shard-pack-manifest.json"},
	}

	prompt := ComposeCollectArtifactPairRepairPrompt(acpruntime.ProviderCodexCode, task, fmt.Errorf("repo evidence path %q is missing under resolved repo root", "missing-evidence.md"))
	docTarget := filepath.Join(writeRoot, "payments-overview.md")
	manifestTarget := filepath.Join(writeRoot, "shard-pack-manifest.json")
	expectedTokens := []string{
		fmt.Sprintf("Write exactly two files in the first command: %q and %q.", docTarget, manifestTarget),
		"If the authored document target already exists, rewrite it completely from observed evidence",
		"do not leave stale missing-path claims in place",
		fmt.Sprintf("exact authored document target = %q", docTarget),
		`"path": "payments-overview.md"`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected existing-doc pair repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if strings.Contains(prompt, `"path": "root-overview.md"`) {
		t.Fatalf("pair repair prompt should target existing authored markdown, got:\n%s", prompt)
	}
	if strings.Contains(prompt, `exact authored document target = "`+filepath.Join(writeRoot, "aaa-general.md")+`"`) {
		t.Fatalf("pair repair prompt should not pick unrelated authored markdown before stale markdown, got:\n%s", prompt)
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

func TestComposeValidatorVerdictRepairPromptIncludesMultiRepoSkeletonSignal(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for _, repo := range []string{"course-discovery", "frontend-platform"} {
		repoDir := filepath.Join(repoRoot, repo)
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", repo, err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte(repo+"\n"), 0o644); err != nil {
			t.Fatalf("write %s readme: %v", repo, err)
		}
	}
	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "refresh.step3.findings",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/validator",
		ReadContextRoots:  []string{filepath.Join(repoRoot, "course-discovery"), filepath.Join(repoRoot, "frontend-platform")},
		RepoScopes:        []string{"course-discovery", "frontend-platform"},
		ExpectedArtifacts: []string{"validator-verdict.json"},
	}

	prompt := ComposeValidatorVerdictRepairPrompt(acpruntime.ProviderQwenCode, task, os.ErrInvalid)
	required := []string{
		"FIRST VALIDATOR VERDICT COMMAND:",
		`"id": "finding.cross_repo.semantic_signal.required"`,
		`"related_ids": [`,
		`"course-discovery"`,
		`"frontend-platform"`,
		`"id": "q.cross_repo.integration_contract"`,
		`"issues": []`,
	}
	for _, token := range required {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected validator repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if got := strings.Count(prompt, "FIRST VALIDATOR VERDICT COMMAND:"); got != 1 {
		t.Fatalf("expected one validator first-action command heading, got %d:\n%s", got, prompt)
	}
	if got := strings.Count(prompt, "finding.cross_repo.semantic_signal.required"); got != 1 {
		t.Fatalf("expected cross-repo finding only inside first heredoc, got %d:\n%s", got, prompt)
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
		"Do not begin with broad analysis. Run the exact shell command below as a single command",
		"overwrite them from the heredoc artifacts",
		"FIRST AS-IS DRAFT COMMAND:",
		"Run this exact shell command as the next filesystem action; it writes asis-draft-manifest.json plus overview.md, summary.md, and architect-summary.md",
		"write_root='/tmp/workspace/reports/taskruns/run-1/asis'",
		"draft_root='/tmp/workspace/reports/taskruns/run-1/staging/final'",
		"cat > \"$write_root/asis-draft-manifest.json\" <<'ACP_DRAFT_MANIFEST_JSON'",
		"cat > \"$draft_root/overview.md\" <<'ACP_DRAFT_FILE'",
		"cat > \"$draft_root/summary.md\" <<'ACP_DRAFT_FILE'",
		"cat > \"$draft_root/architect-summary.md\" <<'ACP_DRAFT_FILE'",
		"test -s \"$write_root/asis-draft-manifest.json\"",
		"test -s \"$draft_root/overview.md\"",
		"test -s \"$draft_root/summary.md\"",
		"test -s \"$draft_root/architect-summary.md\"",
		"first bootstrap draft set",
		"The heredoc as-is files are bootstrap-only repair targets, not valid final content.",
		"replace recovery scaffold text with evidence-backed as-is content",
		"no referenced draft file contains unchanged bootstrap/recovery scaffold",
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
	for _, forbidden := range []string{
		"Provider focused recovery wrote",
		"Provider focused recovery produced",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("draft repair prompt must not write provider-placeholder text %q:\n%s", forbidden, prompt)
		}
	}
	if strings.Count(prompt, "FIRST AS-IS DRAFT COMMAND:") != 1 {
		t.Fatalf("draft repair prompt must contain one as-is first command heading:\n%s", prompt)
	}
	if strings.Count(prompt, "ACP_DRAFT_FILE") != 6 {
		t.Fatalf("draft repair prompt must contain exactly three draft file heredocs:\n%s", prompt)
	}
}

func TestComposeDraftArtifactEnrichmentPromptAvoidsBootstrapHeredoc(t *testing.T) {
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

	prompt := ComposeDraftArtifactEnrichmentPrompt(acpruntime.ProviderClaudeCode, task, os.ErrNotExist)
	for _, token := range []string{
		"draft artifact enrichment focused recovery mode",
		"Immediate draft artifact enrichment action:",
		"Do not run the earlier heredoc/bootstrap draft command again.",
		"Do not answer with a plan, status note, or analysis-only message",
		"Your next action must be a filesystem command that rewrites every referenced markdown draft target",
		"Forbidden analysis-only phrases before the rewrite: I have enough evidence",
		"First focused work unit: execute one bounded filesystem command that reads the current draft manifest",
		"execute one bounded filesystem command",
		"rewrites every referenced markdown target in that same command",
		"bounded staged evidence",
		"bounded_read_context_roots",
		"Fresh mutation is required",
		"rewrite every markdown target",
		"Prefer not to rewrite the draft manifest during enrichment",
		"The only allowed output object keys are path, canonical_path, kind, and title",
		"Never add logical_path",
		"Preserve outputs[].path and outputs[].canonical_path exactly",
		"update only top-level summary or updated_at",
		"/tmp/workspace/reports/taskruns/run-1/asis/asis-draft-manifest.json",
		"/tmp/workspace/reports/taskruns/run-1/staging/final",
		"overview.md -> reports/as-is/overview.md",
		"summary.md -> reports/coverage/summary.md",
		"architect-summary.md -> reports/agent-outputs/architect/summary.md",
		"STEP2 WRITE-FIRST SEQUENCE",
		"your next filesystem command must read asis-draft-manifest.json",
		"all available shard-pack-manifest.json summaries",
		"final-run-index.json and citation-index.json if present",
		"at most 6 high-signal shard manifests or authored shard docs",
		"/tmp/workspace/reports/taskruns/run-1/staging/final/overview.md",
		"/tmp/workspace/reports/taskruns/run-1/staging/final/summary.md",
		"/tmp/workspace/reports/taskruns/run-1/staging/final/architect-summary.md",
		"overview.md must contain: architecture surface summary",
		"summary.md must contain: planned/succeeded/failed shard completeness",
		"architect-summary.md must contain: decision-ready operator summary",
		"Do not stop after writing only one markdown target",
		"Do not stop after saying you have enough evidence",
		"all three step2 markdown targets must be freshly overwritten",
		"If staged evidence is sparse, write the exact missing staged surface",
		"Final self-check: overview.md, summary.md, and architect-summary.md were freshly overwritten",
		"Final markdown must read as an operator-facing architecture/report/proposal artifact",
		"Final content MUST NOT include these scaffold/recovery markers:",
		"This draft is grounded in the current step manifest",
		"bounded staged evidence",
		"recovery pass",
		"Drafted required runtime artifacts for this step",
		"Provider wrote this draft artifact",
		"placeholder content",
		"replace placeholder",
		"Read the current draft manifest only for contract fields and exact outputs",
		"do not quote or copy its bootstrap summary",
		"Runtime draft recovery initialized",
		"Use collected shard manifests and validator output as the evidence source before final review",
		"Do not read every staged shard document",
		"at most 6 authored markdown docs",
		"A no-op rewrite is invalid",
		"Final markdown must summarize structured JSON evidence in readable prose or compact bullets.",
		"every inline-code/path backtick pair must be balanced",
		"Do not copy raw authored-shard prose fragments that contain backticks",
		"Do not end prose sentences with stray backticks",
		"Do not paste raw JSON, Python dict/list reprs",
		"Enrich overview.md, summary.md, and architect-summary.md from collected shard manifests, bounded authored shard docs",
		"decision-ready operator summary",
		"coverage gaps",
		"For shard completeness, derive planned/succeeded/failed from typed shard-plan/shard-summary artifacts when visible",
		"Never count the words failed/error/summary lexically inside manifests or markdown",
		"If planned shard status is not explicitly visible, write planned=unknown",
		"Do not list final-run-index.json or citation-index.json from a different run_id as current-run evidence",
		"final-run-index.json and citation-index.json are downstream/final staging artifacts and may not exist yet during step2",
		"If they are absent, omit final-index availability from the as-is markdown",
		"do not write that current-run final/citation indexes are missing, not found, or unavailable",
		"If final-run-index.json or citation-index.json are present for current_run_id, summarize counts",
		"Do not paste raw object payloads, `documents=[{...}]`, `citations=[{...}]`, or Python-style dict snippets.",
		"Do not write broken path bullets such as a lone backtick",
		"Do not paste sampled authored-shard snippets as semicolon-separated prose",
		"Translate runtime evidence into architecture facts, coverage gaps, and operator decisions",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected draft enrichment prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	for _, forbidden := range []string{
		"ACP_DRAFT_FILE",
		"ACP_DRAFT_MANIFEST_JSON",
		"cat >",
		"FIRST AS-IS DRAFT COMMAND:",
		"Runtime draft recovery initialized this artifact for the scoped analysis step.",
		`"logical_path"`,
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("draft enrichment prompt must not contain bootstrap heredoc/scaffold %q:\n%s", forbidden, prompt)
		}
	}
}

func TestComposeDraftArtifactEnrichmentPromptNamesShardStatusEvidenceAndTargetIdentity(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	taskrunsRoot := filepath.Join(workspace, "reports", "taskruns")
	if err := os.MkdirAll(taskrunsRoot, 0o755); err != nil {
		t.Fatalf("mkdir taskruns root: %v", err)
	}
	planPath := filepath.Join(taskrunsRoot, "run-1-refresh-step1-collect-shard-plan-ftgo-application.json")
	summaryPath := filepath.Join(taskrunsRoot, "run-1-refresh-step1-collect-shard-summary-ftgo-application.json")
	if err := os.WriteFile(planPath, []byte(`{"items":[{"shard_id":"a"}]}`), 0o644); err != nil {
		t.Fatalf("write shard plan: %v", err)
	}
	if err := os.WriteFile(summaryPath, []byte(`{"items":[{"status":"succeeded"},{"status":"failed"}]}`), 0o644); err != nil {
		t.Fatalf("write shard summary: %v", err)
	}

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "refresh.step2.asis_docs",
		StepContract:      "as_is",
		DomainID:          "ftgo-application",
		RepoScope:         "ftgo-application",
		RepoScopes:        []string{"ftgo-application"},
		AgentRole:         "architect",
		Workspace:         workspace,
		WriteRoot:         filepath.Join(workspace, "reports", "taskruns", "run-1", "runtime", "step2_as_is"),
		DraftFinalRoot:    filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "drafts", "step2_as_is"),
		ExpectedArtifacts: []string{"asis-draft-manifest.json", "overview.md", "summary.md", "architect-summary.md"},
	}

	prompt := ComposeDraftArtifactEnrichmentPrompt(acpruntime.ProviderCodexCode, task, os.ErrInvalid)
	for _, token := range []string{
		"DRAFT ENRICHMENT TARGET IDENTITY:",
		`current_run_id = "run-1"`,
		`current_step_id = "refresh.step2.asis_docs"`,
		`current_domain_id = "ftgo-application"`,
		`current_repo_scope = "ftgo-application"`,
		"current_repo_scopes = ftgo-application",
		"Current target identity comes from repo_scope/repo_scopes/domain_id",
		"not from matrix id, batch id, profile id, workspace path, or run-folder names",
		"final markdown must not name sibling matrix targets or other repositories unless an allowed staged artifact",
		"Matrix/profile/batch names such as combined multi-target folder names are harness labels, not architecture evidence.",
		"Final markdown must not cite taskrun identifiers other than current_run_id",
		"DRAFT ENRICHMENT CURRENT-RUN SHARD STATUS EVIDENCE:",
		"Read these exact current-run typed shard-plan/shard-summary files",
		planPath,
		summaryPath,
		"planned = len(items)",
		"status == \"succeeded\"",
		"status == \"failed\"",
		"Do not report planned=unknown or failed=unknown when a readable current-run typed shard-summary items[] list is available.",
		"When a readable typed shard-summary shows failed=0 and no pending/checkpointed/other statuses",
		"write exact current-run counts and an explicit no-shard-coverage-blocker statement",
		"Shard completeness: 16/16 succeeded; no failed, pending, or incomplete shard statuses were observed in the current-run typed shard summary.",
		`"failed shards require rerun"`,
		"current-run typed shard-plan/shard-summary files listed above when present",
		"including shard-summary items[].status",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected draft enrichment prompt to contain %q, got:\n%s", token, prompt)
		}
	}
}

func TestComposeDraftArtifactEnrichmentPromptForProposalsRequiresWriteFirstTargets(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		StepContract:      "proposals",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/proposals",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ExpectedArtifacts: []string{"proposals-draft-manifest.json", "proposal.md", "changelog.md"},
	}

	prompt := ComposeDraftArtifactEnrichmentPrompt(acpruntime.ProviderClaudeCode, task, os.ErrInvalid)
	for _, token := range []string{
		"draft artifact enrichment focused recovery mode",
		"First focused work unit: execute one bounded filesystem command that reads the current draft manifest",
		"rewrites every referenced markdown target in that same command",
		"proposal.md -> proposals/runtime-recommendations.md",
		"changelog.md -> reports/changelog/runtime-proposals.md",
		"STEP4 WRITE-FIRST SEQUENCE",
		"read proposals-draft-manifest.json",
		"current-run typed shard-plan/shard-summary files listed above when present",
		"final-run-index.json and citation-index.json if present",
		"at most 6 high-signal shard manifests or authored shard docs",
		"/tmp/workspace/reports/taskruns/run-1/staging/final/proposal.md",
		"/tmp/workspace/reports/taskruns/run-1/staging/final/changelog.md",
		"proposal.md must contain: Decision / recommended operator action",
		"Evidence used with repo/path or staged artifact references",
		"Risks, gaps, and out-of-scope notes",
		"changelog.md must contain: Updated architecture/proposal surfaces",
		"Evidence index or citation references",
		"Residual coverage gaps",
		"Do not report 0 authored markdown shard documents unless you actually globbed staging/shards/**/*.md",
		"Do not ask the operator to re-run or repair non-succeeded shards when the current-run typed shard-summary shows failed=0",
		"write exact planned/succeeded/failed/incomplete counts plus an explicit no-shard-coverage-blocker statement in both proposal.md and changelog.md instead.",
		"Do not list final-run-index.json, citation-index.json, validator verdicts, or shard summaries from a different run_id",
		"Final markdown must summarize structured JSON evidence in readable prose or compact bullets.",
		"When final-run-index.json or citation-index.json are present for current_run_id, summarize counts",
		"Do not write stale index availability claims such as `No current-run final-run-index document list was available`",
		"Do not write stale zero-count claims such as `final-run-index.json contains 0 observed document entries`",
		"Do not paste raw object payloads, Python-style dict snippets, `{'id': ...}`, or truncated JSON fragments.",
		"Do not paste sampled shard markdown snippets with inline-code markers into proposal.md or changelog.md",
		"Do not claim citation detail is limited or unavailable when current-run citation-index.json contains citation entries.",
		"Do not mention placeholder replacement, placeholder proposal content, replaced placeholder content, or recovery mechanics",
		"placeholder proposal content",
		"replaced placeholder content",
		"If staged evidence is sparse, write the gap explicitly",
		"Do not treat proposals-draft-manifest.json summary text, canonical_path examples, or bootstrap output metadata as findings/proposals",
		"record an explicit no-actionable-proposal gap",
		"Final self-check: both proposal.md and changelog.md were freshly overwritten",
		"Final markdown must read as an operator-facing architecture/report/proposal artifact",
		"Final content MUST NOT include these scaffold/recovery markers:",
		"current draft manifest",
		"enrichment read",
		"Drafted required runtime artifacts for this step",
		"Runtime proposal surface initialized",
		"Convert findings into concrete operator decisions",
		"Previous draft artifact validation failure:",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected proposals draft enrichment prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	for _, forbidden := range []string{
		"ACP_DRAFT_FILE",
		"ACP_DRAFT_MANIFEST_JSON",
		"cat >",
		"FIRST PROPOSALS DRAFT COMMAND:",
		"Runtime proposal surface initialized for this analysis run.",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("proposals draft enrichment prompt must not contain bootstrap heredoc/scaffold %q:\n%s", forbidden, prompt)
		}
	}
}

func TestComposeDraftArtifactEnrichmentPromptAddsMarkdownSyntaxRetryHint(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		StepContract:      "proposals",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/proposals",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ExpectedArtifacts: []string{"proposals-draft-manifest.json", "proposal.md", "changelog.md"},
	}
	err := errors.New(`runtime draft manifest outputs are invalid: outputs[0].path "proposal.md" contains malformed markdown inline-code or code-fence syntax`)

	prompt := ComposeDraftArtifactEnrichmentPrompt(acpruntime.ProviderCodexCode, task, err)
	for _, token := range []string{
		"DRAFT ENRICHMENT MARKDOWN SYNTAX RETRY:",
		"The previous enrichment attempt failed because at least one referenced markdown file had malformed inline-code or code-fence syntax.",
		"Preserve the evidence-backed meaning, but remove or balance all inline backticks before exit.",
		"Prefer plain text service/module names over inline-code when summarizing sampled shard prose.",
		"Do not copy truncated shard excerpts, raw snippets, or semicolon lists that may carry half-open backticks.",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected markdown syntax retry prompt to contain %q, got:\n%s", token, prompt)
		}
	}
}

func TestComposeArtifactOnlyPromptIncludesQAFirstActionAndPromptPack(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	packDir := filepath.Join(workspaceDir, "skills", "prompt-packs")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("mkdir prompt pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "qa.md"), []byte("Use concise workspace-backed answers.\n"), 0o644); err != nil {
		t.Fatalf("write qa prompt pack: %v", err)
	}

	task := acpruntime.Task{
		RunID:             "run-qa-1",
		StepID:            acpruntime.StepIDQAAsk,
		Workspace:         workspaceDir,
		WriteRoot:         filepath.Join(workspaceDir, "reports", "taskruns", "run-qa-1", "qa"),
		Question:          "Who owns payments-service?",
		ContextPackPath:   filepath.Join(workspaceDir, "reports", "taskruns", "run-qa-1", "qa", "context-pack.json"),
		ExpectedArtifacts: []string{"qa-answer.json"},
	}

	prompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)
	for _, needle := range []string{
		`FIRST QA ANSWER COMMAND:`,
		`STEP POLICY qa.ask:`,
		`Source file: "skills/prompt-packs/qa.md"`,
		`Use concise workspace-backed answers.`,
		`qa-answer.json`,
	} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("expected qa prompt to contain %q, got:\n%s", needle, prompt)
		}
	}
}

func TestComposeDraftArtifactRepairPromptWritesValidConstitutionSubagentsYaml(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/constitution",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
	}

	prompt := ComposeDraftArtifactRepairPrompt(acpruntime.ProviderClaudeCode, task, os.ErrNotExist)
	for _, token := range []string{
		"FIRST CONSTITUTION DRAFT COMMAND:",
		"first bootstrap draft set",
		"write_root='/tmp/workspace/reports/taskruns/run-1/constitution'",
		"draft_root='/tmp/workspace/reports/taskruns/run-1/staging/final'",
		"cat > \"$draft_root/baseline-subagents.yaml\" <<'ACP_DRAFT_FILE'",
		"test -s \"$draft_root/baseline-subagents.yaml\"",
		"The heredoc charter overview is a bootstrap-only repair target, not valid final content.",
		"replace recovery scaffold text in charter-overview.md with evidence-backed constitution content",
		"charter-overview.md has no unchanged bootstrap/recovery scaffold",
		"agents:",
		"id: domain-analyst",
		"id: architect-aggregator",
		"id: system-analyst-qa",
	} {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected constitution draft repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if strings.Contains(prompt, "generated_by: acp-runtime-provider-focused-recovery") {
		t.Fatalf("constitution draft repair must write canonical subagents YAML, not generic YAML:\n%s", prompt)
	}
}

func TestComposeDraftArtifactRepairPromptAvoidsRetypingLongDraftPaths(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:          "run-1",
		StepID:         "init.step0.constitution",
		StepContract:   "constitution",
		WriteRoot:      "/tmp/e2e/frontend/claude-code/frontend-workspace/reports/taskruns/run-1/runtime/step0_constitution",
		DraftFinalRoot: "/tmp/e2e/frontend/claude-code/frontend-workspace/reports/taskruns/run-1/staging/drafts/step0_constitution",
	}

	prompt := ComposeDraftArtifactRepairPrompt(acpruntime.ProviderClaudeCode, task, os.ErrNotExist)
	required := []string{
		"Do not manually retype or transform absolute paths; keep slash-separated path components unchanged.",
		"write_root='/tmp/e2e/frontend/claude-code/frontend-workspace/reports/taskruns/run-1/runtime/step0_constitution'",
		"draft_root='/tmp/e2e/frontend/claude-code/frontend-workspace/reports/taskruns/run-1/staging/drafts/step0_constitution'",
		"cat > \"$draft_root/baseline-subagents.yaml\" <<'ACP_DRAFT_FILE'",
		"test -s \"$draft_root/baseline-subagents.yaml\"",
	}
	for _, token := range required {
		if !strings.Contains(prompt, token) {
			t.Fatalf("expected draft repair prompt to contain %q, got:\n%s", token, prompt)
		}
	}
	if strings.Contains(prompt, "frontend-claude-code") {
		t.Fatalf("draft repair prompt must not collapse slash-separated provider path components:\n%s", prompt)
	}
	if got := strings.Count(prompt, "/tmp/e2e/frontend/claude-code/frontend-workspace/reports/taskruns/run-1/staging/drafts/step0_constitution"); got != 3 {
		t.Fatalf("expected draft root absolute path to appear only in policy lines and variable assignment, got %d:\n%s", got, prompt)
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

func TestComposeArtifactOnlyPromptAddsValidatorCrossRepoFirstActionSignal(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for _, repo := range []string{"course-discovery", "frontend-platform"} {
		repoDir := filepath.Join(repoRoot, repo)
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", repo, err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte(repo+"\n"), 0o644); err != nil {
			t.Fatalf("write %s readme: %v", repo, err)
		}
	}
	task := acpruntime.Task{
		StepID:            "refresh.step3.findings",
		WriteRoot:         "/tmp/write-root",
		ReadContextRoots:  []string{filepath.Join(repoRoot, "course-discovery"), filepath.Join(repoRoot, "frontend-platform")},
		RepoScopes:        []string{"course-discovery", "frontend-platform"},
		ExpectedArtifacts: []string{"validator-verdict.json"},
	}

	claudePrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
	qwenPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)
	codexPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)
	claudeTail := strings.TrimPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\n")
	qwenTail := strings.TrimPrefix(qwenPrompt, "You are ACP runtime provider \"qwen-code\".\n\n")
	codexTail := strings.TrimPrefix(codexPrompt, "You are ACP runtime provider \"codex-code\".\n\n")
	if claudeTail != qwenTail || claudeTail != codexTail {
		t.Fatalf("expected validator providers to share identical multi-repo enforced prompt body")
	}

	for _, token := range []string{
		"VALIDATOR FIRST-ACTION ARTIFACT:",
		"FIRST VALIDATOR VERDICT COMMAND:",
		`"id": "finding.cross_repo.semantic_signal.required"`,
		`"id": "q.cross_repo.integration_contract"`,
		`"repo": "course-discovery"`,
		`"repo": "frontend-platform"`,
		"Artifact-only contract:",
	} {
		if !strings.Contains(claudePrompt, token) {
			t.Fatalf("expected validator prompt to contain %q, got:\n%s", token, claudePrompt)
		}
	}
	if strings.Index(claudePrompt, "finding.cross_repo.semantic_signal.required") > strings.Index(claudePrompt, "Artifact-only contract:") {
		t.Fatalf("expected cross-repo skeleton in first-action section before artifact-only contract:\n%s", claudePrompt)
	}
	if got := strings.Count(claudePrompt, "finding.cross_repo.semantic_signal.required"); got != 1 {
		t.Fatalf("expected cross-repo finding only inside first heredoc, got %d:\n%s", got, claudePrompt)
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

	claudePrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
	qwenPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)
	codexPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)

	claudeTail := strings.TrimPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\n")
	qwenTail := strings.TrimPrefix(qwenPrompt, "You are ACP runtime provider \"qwen-code\".\n\n")
	codexTail := strings.TrimPrefix(codexPrompt, "You are ACP runtime provider \"codex-code\".\n\n")
	if claudeTail != qwenTail || claudeTail != codexTail {
		t.Fatalf("expected as-is enforced prompt body to be provider-independent")
	}

	expectedTokens := []string{
		"AS-IS FIRST-ACTION DRAFT ARTIFACTS:",
		"FIRST AS-IS DRAFT COMMAND:",
		"write_root='/tmp/write-root'",
		"draft_root='/tmp/draft-root'",
		"cat > \"$write_root/asis-draft-manifest.json\" <<'ACP_DRAFT_MANIFEST_JSON'",
		"cat > \"$draft_root/overview.md\" <<'ACP_DRAFT_FILE'",
		"cat > \"$draft_root/summary.md\" <<'ACP_DRAFT_FILE'",
		"cat > \"$draft_root/architect-summary.md\" <<'ACP_DRAFT_FILE'",
		"The first draft artifact set is bootstrap-only",
		"replace placeholder scaffold text with evidence-backed as-is content",
		"AS-IS DRAFT MANIFEST CANONICAL SHAPE:",
		`asis-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="as_is", agent_role, and outputs[]; optional top-level metadata is limited to summary and updated_at.`,
		`overview.md -> reports/as-is/overview.md`,
		`"step_contract": "as_is"`,
		`"canonical_path": "reports/as-is/payments/overview.md"`,
	}
	for _, token := range expectedTokens {
		if !strings.Contains(claudePrompt, token) {
			t.Fatalf("expected as-is prompt to contain %q, got:\n%s", token, claudePrompt)
		}
	}
	if !strings.HasPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\nAS-IS FIRST-ACTION DRAFT ARTIFACTS:") {
		t.Fatalf("expected as-is first-action section immediately after provider identity, got:\n%s", claudePrompt)
	}
	firstActionIndex := strings.Index(claudePrompt, "FIRST AS-IS DRAFT COMMAND:")
	artifactContractIndex := strings.Index(claudePrompt, "Artifact-only contract:")
	if firstActionIndex < 0 || artifactContractIndex < 0 || firstActionIndex > artifactContractIndex {
		t.Fatalf("expected as-is first-action command before broad artifact-only contract:\n%s", claudePrompt)
	}
	if got := strings.Count(claudePrompt, "FIRST AS-IS DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected one as-is first-action command heading, got %d:\n%s", got, claudePrompt)
	}
	if got := strings.Count(claudePrompt, "ACP_DRAFT_MANIFEST_JSON"); got != 2 {
		t.Fatalf("expected one as-is manifest heredoc in normal prompt, got delimiter count %d:\n%s", got, claudePrompt)
	}
	if got := strings.Count(claudePrompt, "ACP_DRAFT_FILE"); got != 6 {
		t.Fatalf("expected three as-is draft file heredocs in normal prompt, got delimiter count %d:\n%s", got, claudePrompt)
	}
}

func TestComposeArtifactOnlyPromptAddsConstitutionFirstActionCommand(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/write-root",
		DraftFinalRoot:    "/tmp/draft-root",
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
	}

	claudePrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
	qwenPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)
	codexPrompt := ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)

	claudeTail := strings.TrimPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\n")
	qwenTail := strings.TrimPrefix(qwenPrompt, "You are ACP runtime provider \"qwen-code\".\n\n")
	codexTail := strings.TrimPrefix(codexPrompt, "You are ACP runtime provider \"codex-code\".\n\n")
	if claudeTail != qwenTail || claudeTail != codexTail {
		t.Fatalf("expected constitution enforced prompt body to be provider-independent")
	}

	expectedTokens := []string{
		"CONSTITUTION FIRST-ACTION DRAFT ARTIFACTS:",
		"FIRST CONSTITUTION DRAFT COMMAND:",
		"write_root='/tmp/write-root'",
		"draft_root='/tmp/draft-root'",
		"cat > \"$write_root/constitution-draft.json\" <<'ACP_DRAFT_MANIFEST_JSON'",
		"cat > \"$draft_root/charter-overview.md\" <<'ACP_DRAFT_FILE'",
		"cat > \"$draft_root/baseline-subagents.yaml\" <<'ACP_DRAFT_FILE'",
		"The heredoc charter overview is bootstrap-only",
		"Artifact-only contract:",
		`"step_id": "init.step0.constitution"`,
		`"step_contract": "constitution"`,
		`"canonical_path": "charter/overview.md"`,
		`"canonical_path": "skills/subagents.yaml"`,
		"replace placeholder scaffold text in charter-overview.md with evidence-backed charter content",
		"stop only after confirming charter-overview.md is not an unchanged bootstrap placeholder",
		"constitution-draft.json must use the exact runtime draft manifest shape shown below",
	}
	for _, token := range expectedTokens {
		if !strings.Contains(claudePrompt, token) {
			t.Fatalf("expected constitution prompt to contain %q, got:\n%s", token, claudePrompt)
		}
	}
	if !strings.HasPrefix(claudePrompt, "You are ACP runtime provider \"claude-code\".\n\nCONSTITUTION FIRST-ACTION DRAFT ARTIFACTS:") {
		t.Fatalf("expected constitution first-action section immediately after provider identity, got:\n%s", claudePrompt)
	}
	firstActionIndex := strings.Index(claudePrompt, "FIRST CONSTITUTION DRAFT COMMAND:")
	artifactContractIndex := strings.Index(claudePrompt, "Artifact-only contract:")
	if firstActionIndex < 0 || artifactContractIndex < 0 || firstActionIndex > artifactContractIndex {
		t.Fatalf("expected constitution first-action command before broad artifact-only contract:\n%s", claudePrompt)
	}
	if got := strings.Count(claudePrompt, "FIRST CONSTITUTION DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected one constitution first-action command heading, got %d:\n%s", got, claudePrompt)
	}
	if got := strings.Count(claudePrompt, "ACP_DRAFT_MANIFEST_JSON"); got != 2 {
		t.Fatalf("expected one constitution manifest heredoc in normal prompt, got delimiter count %d:\n%s", got, claudePrompt)
	}
	if got := strings.Count(claudePrompt, "ACP_DRAFT_FILE"); got != 4 {
		t.Fatalf("expected two constitution draft file heredocs in normal prompt, got delimiter count %d:\n%s", got, claudePrompt)
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
		"write_root='/tmp/write-root'",
		"draft_root='/tmp/draft-root'",
		"cat > \"$write_root/proposals-draft-manifest.json\" <<'ACP_DRAFT_MANIFEST_JSON'",
		"cat > \"$draft_root/proposal.md\" <<'ACP_DRAFT_FILE'",
		"cat > \"$draft_root/changelog.md\" <<'ACP_DRAFT_FILE'",
		"The first proposals draft artifact set is bootstrap-only",
		"replace placeholder scaffold text with evidence-backed proposal/changelog content",
		"PROPOSALS DRAFT MANIFEST CANONICAL SHAPE:",
		`proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="proposals", agent_role, outputs[], and optional summary/updated_at.`,
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
