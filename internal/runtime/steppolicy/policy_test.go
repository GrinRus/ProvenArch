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
		`Default TaskResult template for this step is changeset: []; do not invent synthetic upsert_entity operations for draft-only output.`,
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
		`If authored docs already exist in write_root, respond immediately with the final TaskResult JSON object.`,
		`documents[].path MUST be artifact_root-relative only`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected collect doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestCollectArtifactRepairHintsBanLegacyRepairSurface(t *testing.T) {
	t.Parallel()

	hints := strings.Join(CollectArtifactRepairHints("manifest drift"), "\n")
	if !strings.Contains(hints, `Do NOT describe shard-pack-manifest.json repair via add_doc_artifact; repair the file in write_root and return "changeset": [].`) {
		t.Fatalf("expected collect repair hints to ban legacy add_doc_artifact repair surface, got:\n%s", hints)
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
	if !strings.Contains(hints, `Do NOT describe draft manifest repair via add_doc_artifact when the draft manifest already describes the publish surface.`) {
		t.Fatalf("expected draft repair hints to ban legacy add_doc_artifact repair surface, got:\n%s", hints)
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
