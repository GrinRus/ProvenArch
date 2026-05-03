package steppolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestStepSpecificPolicyDefinesSharedDraftOnlyObligationsForStep0(t *testing.T) {
	t.Parallel()

	policy := StepSpecificPolicy("init.step0.constitution")
	required := []string{
		`Do NOT delegate to agent/subagent helpers.`,
		`Do NOT use todo_write-style planning or long plan narration.`,
		`constitution-draft.json must use the runtime draft manifest contract exactly; legacy constitution schemas are forbidden.`,
		`This is a draft-only step; do not invent semantic entities, edges, findings, or questions on stdout.`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected step0 policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyDefinesSharedCollectRepairSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        "/tmp/workspace",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/payments",
		ReadContextRoots: []string{"/tmp/repos/payments"},
		RepoScopes:       []string{"payments"},
		PathScopes:       []string{"payments"},
		ShardID:          "payments",
		ExpectedArtifacts: []string{
			"shard-pack-manifest.json",
		},
		StartedAtUTC: time.Date(2026, 4, 21, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write ONLY inside write_root.`,
		`Suggested collect authored doc path for this shard:`,
		`Early pair-write requirement: write the suggested overview doc and shard-pack-manifest.json as one focused artifact pair`,
		`Minimal collect target shape: write "payments-overview.md" + "shard-pack-manifest.json" early`,
		`Do not wait for a complete broad repository sweep before writing shard-pack-manifest.json`,
		`FIRST COLLECT ARTIFACT PAIR COMMAND:`,
		`Run this exact command as the next filesystem action after checking whether both target files already exist`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md' <<'ACP_COLLECT_DOC'`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json' <<'ACP_MANIFEST_JSON'`,
		`TASK-SPECIFIC COLLECT MANIFEST JSON SKELETON: use the heredoc JSON embedded in FIRST COLLECT ARTIFACT PAIR COMMAND above`,
		`"path": "payments-overview.md"`,
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
		`COLLECT MANIFEST CONTRACT CHECKLIST:`,
		`The task-specific collect manifest JSON skeleton above is normative`,
		`Do not exit after writing markdown only; every collect shard must finish with a valid shard-pack-manifest.json.`,
		`After the first filesystem write inside write_root, stop broad repository exploration; only minimal manifest/JSON repair is allowed afterwards.`,
		`After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
		`Do NOT read reports/taskruns/**, raw runtime logs, or previously generated shard-pack-manifest.json files as schema examples during collect.`,
		`If authored docs and shard-pack-manifest.json already exist in write_root, stop and exit successfully.`,
		`documents[].path MUST be artifact_root-relative only`,
		`semantic.entities[*].provenance.evidence[*], semantic.edges[*].provenance.evidence[*], and semantic.findings[*].provenance.evidence[*] item MUST include non-empty repo and path values`,
		`Citation-only semantic evidence objects are forbidden`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected collect doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyAddsRootFileShardHint(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        "/tmp/workspace",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/staging/shards/root-files",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/root-files",
		ReadContextRoots: []string{"/tmp/repos/bank"},
		RepoScopes:       []string{"bank"},
		PathScopes:       []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
		ExpectedArtifacts: []string{
			"shard-pack-manifest.json",
		},
		StartedAtUTC: time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Root-file collect shard detected: path_scopes contains root-level files only:`,
		`.gitignore, LICENSE, Makefile, README.md, pom.xml`,
		`read only the listed root files first; do not recursively sweep top-level directories`,
		`Produce one concise root overview document in write_root, then write shard-pack-manifest.json`,
		`"path": "root-overview.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected root-file shard policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyDoesNotTreatTopLevelDirsAsRootFileShard(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        "/tmp/workspace",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/staging/shards/source-docs",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/source-docs",
		ReadContextRoots: []string{"/tmp/repos/bank"},
		RepoScopes:       []string{"bank"},
		PathScopes:       []string{"docs", "src"},
		ShardID:          "bank-source-docs",
	}

	policy := DocFirstFilesystemPolicy(task)
	if strings.Contains(policy, "Root-file collect shard detected") {
		t.Fatalf("top-level directory scopes must not receive root-file shard hint:\n%s", policy)
	}
	if !strings.Contains(policy, `"path": "bank-source-docs-overview.md"`) {
		t.Fatalf("expected non-root shard to use shard-based doc suggestion, got:\n%s", policy)
	}

	task.PathScopes = []string{".github", "README.md"}
	task.ShardID = "bank-root-and-ci"
	policy = DocFirstFilesystemPolicy(task)
	if strings.Contains(policy, "Root-file collect shard detected") {
		t.Fatalf("mixed root files and top-level service dirs must not receive root-file shard hint:\n%s", policy)
	}
}

func TestCollectManifestTaskSkeletonParsesAsShardPackManifest(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/source-docs",
		RepoScopes:   []string{"bank"},
		PathScopes:   []string{"src"},
		ShardID:      "bank-source-docs",
		DomainID:     "payments",
		AgentRole:    "shard-analyst",
	}

	raw := CollectManifestTaskSkeleton(task, []string{"docs/overview.md", "overview.md"}, []string{"src/main.go"})
	manifest, err := contracts.ParseShardPackManifest([]byte(raw))
	if err != nil {
		t.Fatalf("expected task skeleton to parse as a valid shard pack manifest, got %v\n%s", err, raw)
	}
	if got, want := len(manifest.Documents), 2; got != want {
		t.Fatalf("documents = %d, want %d in skeleton:\n%s", got, want, raw)
	}
	if got, want := manifest.Citations[0].Path, "src/main.go"; got != want {
		t.Fatalf("citation path = %q, want %q", got, want)
	}
	if strings.Contains(raw, "scaffold") {
		t.Fatalf("collect manifest skeleton should avoid scaffold wording in provider-authored artifacts:\n%s", raw)
	}
}

func TestDocFirstFilesystemPolicyDefinesCanonicalValidatorVerdictSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step3.findings",
		Workspace:         "/tmp/workspace",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/validator",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ReadContextRoots:  []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
		ExpectedArtifacts: []string{"validator-verdict.json"},
		StartedAtUTC:      time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write validator-verdict.json in write_root.`,
		`Absolute validator verdict target: "/tmp/workspace/reports/taskruns/run-1/validator/validator-verdict.json".`,
		`validator-verdict.json MUST include version=1, run_id, generated_at, verdict, and checked_paths.`,
		`issues[] items MUST use exactly the canonical validator issue shape`,
		`Do NOT put legacy finding-shaped fields inside issues[]`,
		`Canonical validator-verdict fragment below is normative`,
		`"generated_at": "2026-04-16T12:00:02Z"`,
		`"code": "staged_index_missing"`,
		`"repo": "payments-service"`,
		`"verdict": "PASS"`,
		`owner-only residual evidence gaps may still return verdict=PASS when no technical validator issues remain.`,
		`"title": "Owner mapping remains unresolved"`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected findings doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestValidatorVerdictTaskSkeletonParsesWithCanonicalIssueShape(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step3.findings",
		ReadContextRoots: []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
	}

	raw := ValidatorVerdictTaskSkeleton(task)
	verdict, err := contracts.ParseValidatorVerdict([]byte(raw))
	if err != nil {
		t.Fatalf("expected validator verdict skeleton to parse, got %v\n%s", err, raw)
	}
	if verdict.RunID != "run-1" || verdict.Verdict != "PASS" {
		t.Fatalf("unexpected validator verdict skeleton: %+v", verdict)
	}
	if len(verdict.Issues) != 0 {
		t.Fatalf("expected empty canonical issues skeleton, got %+v", verdict.Issues)
	}
}

func TestValidatorVerdictTaskSkeletonUsesStagedFinalRootNotWriteRoot(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:     "run-1",
		StepID:    "init.step3.findings",
		WriteRoot: "/tmp/workspace/reports/taskruns/run-1/validator",
		ReadContextRoots: []string{
			"/tmp/workspace/reports/taskruns/run-1/validator",
			"/tmp/workspace",
			"/tmp/workspace/reports/taskruns/run-1/staging/final",
			"/tmp/repos/bank",
		},
	}

	raw := ValidatorVerdictTaskSkeleton(task)
	verdict, err := contracts.ParseValidatorVerdict([]byte(raw))
	if err != nil {
		t.Fatalf("expected validator verdict skeleton to parse, got %v\n%s", err, raw)
	}
	if len(verdict.CheckedPaths) != 2 {
		t.Fatalf("checked_paths = %#v, want final-run-index and citation-index", verdict.CheckedPaths)
	}
	for _, checkedPath := range verdict.CheckedPaths {
		if strings.Contains(checkedPath, "/validator/") {
			t.Fatalf("validator repair skeleton must not point checked_paths at write_root, got %#v", verdict.CheckedPaths)
		}
		if !strings.Contains(checkedPath, "/staging/final/") {
			t.Fatalf("validator repair skeleton must point checked_paths at staged final artifacts, got %#v", verdict.CheckedPaths)
		}
	}
}

func TestDocFirstFilesystemPolicyDefinesCanonicalAsIsDraftSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step2.asis_docs",
		Workspace:         "/tmp/workspace",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/step2",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ReadContextRoots:  []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
		ExpectedArtifacts: []string{"asis-draft-manifest.json"},
		StartedAtUTC:      time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write asis-draft-manifest.json in write_root.`,
		`Use staged final evidence from read_context_roots only; do NOT read sibling baseline workspaces`,
		`asis-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="as_is", agent_role, and outputs[].`,
		`"step_contract": "as_is"`,
		`"canonical_path": "reports/as-is/overview.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected as-is doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestStepSpecificPolicyDefinesProposalsDraftContract(t *testing.T) {
	t.Parallel()

	policy := StepSpecificPolicy("init.step4.proposals")
	required := []string{
		`STEP POLICY init.step4.proposals:`,
		`Use validated staged final evidence from read_context_roots`,
		`Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only manifest source of truth for proposals-draft-manifest.json.`,
		`Keep step_contract exactly "proposals"`,
		`outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.`,
		`pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected proposals step policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyDefinesCanonicalProposalsDraftSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		Workspace:         "/tmp/workspace",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/step4",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ReadContextRoots:  []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
		StartedAtUTC:      time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write proposals-draft-manifest.json in write_root.`,
		`Allowed canonical targets are proposals/* and reports/changelog/*.`,
		`proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="proposals", agent_role, optional summary, and outputs[].`,
		`Do NOT add legacy top-level fields such as pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.`,
		`"step_contract": "proposals"`,
		`"canonical_path": "proposals/proposal-baseline/proposal.md"`,
		`"canonical_path": "reports/changelog/run-1.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected proposals doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestCollectArtifactRepairHintsBanLegacyRepairSurface(t *testing.T) {
	t.Parallel()

	hints := strings.Join(CollectArtifactRepairHints("manifest drift"), "\n")
	if !strings.Contains(hints, `Repair mode is artifact-only: do not invent extra repository file reads/writes after authored docs already exist.`) {
		t.Fatalf("expected collect repair hints to ban extra repair writes outside the artifact-only surface, got:\n%s", hints)
	}
}

func TestDraftArtifactRepairHintsBanLegacyRepairSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		WriteRoot:         "/tmp/write-root",
		DraftFinalRoot:    "/tmp/draft-root",
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
	}

	hints := strings.Join(DraftArtifactRepairHints(task, nil), "\n")
	if !strings.Contains(hints, `Repair mode is draft-only: do not invent extra repository file reads/writes after draft files already exist.`) {
		t.Fatalf("expected draft repair hints to ban extra repair writes outside the draft-only surface, got:\n%s", hints)
	}
	for _, needle := range []string{
		`step_contract MUST be exactly "proposals"`,
		`outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/* and MUST be unique.`,
		`pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps`,
	} {
		if !strings.Contains(hints, needle) {
			t.Fatalf("expected proposals repair hints to contain %q, got:\n%s", needle, hints)
		}
	}
}

func TestWorkspacePromptPackSectionLoadsEditableContentLayer(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	packPath := filepath.Join(workspaceDir, "skills", "prompt-packs")
	if err := os.MkdirAll(packPath, 0o755); err != nil {
		t.Fatalf("mkdir prompt pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packPath, "collect-context.md"), []byte("Collect pack line A\nCollect pack line B\n"), 0o644); err != nil {
		t.Fatalf("write prompt pack: %v", err)
	}

	section := WorkspacePromptPackSection(acpruntime.Task{
		StepID:    "init.step1.collect",
		Workspace: workspaceDir,
	})
	required := []string{
		`WORKSPACE PROMPT PACK CONTENT LAYER:`,
		`Source file: "skills/prompt-packs/collect-context.md"`,
		`editable content layer only`,
		`Collect pack line A`,
		`END WORKSPACE PROMPT PACK`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected workspace prompt-pack section to contain %q, got:\n%s", needle, section)
		}
	}
}

func TestCollectRepoEntrypointHintsIncludesOwnershipFiles(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".github"), 0o755); err != nil {
		t.Fatalf("mkdir .github: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".github", "CODEOWNERS"), []byte("* @platform\n"), 0o644); err != nil {
		t.Fatalf("write codeowners: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "CODEOWNERS"), []byte("* @root-owners\n"), 0o644); err != nil {
		t.Fatalf("write root codeowners: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "OWNERS.md"), []byte("# Owners\n"), 0o644); err != nil {
		t.Fatalf("write owners: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "MAINTAINERS.yaml"), []byte("maintainers: []\n"), 0o644); err != nil {
		t.Fatalf("write maintainers: %v", err)
	}

	hints := CollectRepoEntrypointHints(acpruntime.Task{
		StepID:           "init.step1.collect",
		ReadContextRoots: []string{repoRoot},
	})
	joined := strings.Join(hints, "\n")
	for _, needle := range []string{".github/CODEOWNERS", "CODEOWNERS", "OWNERS.md", "MAINTAINERS.yaml"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected entrypoint hints to contain %q, got:\n%s", needle, joined)
		}
	}
}
