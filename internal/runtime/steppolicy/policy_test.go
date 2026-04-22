package steppolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		ExpectedArtifacts: []string{
			"shard-pack-manifest.json",
		},
		StartedAtUTC: time.Date(2026, 4, 21, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write ONLY inside write_root.`,
		`After the first filesystem write inside write_root, stop broad repository exploration; only minimal manifest/JSON repair is allowed afterwards.`,
		`After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
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
		`validator-verdict.json MUST include version=1, run_id, generated_at, verdict, and checked_paths.`,
		`Canonical validator-verdict fragment below is normative`,
		`"generated_at": "2026-04-16T12:00:02Z"`,
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
