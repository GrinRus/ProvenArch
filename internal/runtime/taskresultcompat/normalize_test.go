package taskresultcompat

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestNormalizeRawTaskResultDropsLegacyManifestRepairOp(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/acp/write-root",
  "documents": [
    {
      "id": "doc.1",
      "kind": "analysis",
      "title": "Analysis",
      "path": "iac-overview.md",
      "canonical_path": "reports/as-is/iac-overview.md",
      "topics": ["iac"],
      "citation_ids": ["cite.1"]
    }
  ],
  "citations": [
    {
      "id": "cite.1",
      "repo": "bank-of-anthos",
      "path": "iac/README.md",
      "claim_ids": ["claim.1"],
      "document_ids": ["doc.1"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["iac"], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`), 0o644); err != nil {
		t.Fatalf("write valid shard-pack-manifest: %v", err)
	}

	task := acpruntime.Task{
		StepID:    "init.step1.collect",
		WriteRoot: writeRoot,
	}
	raw := []byte(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step1.collect",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-18T21:54:16Z"
  },
  "summary": "manifest repaired",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "artifact": {
        "id": "shard-pack-manifest.json",
        "path": "/tmp/acp/write-root/shard-pack-manifest.json"
      }
    }
  ],
  "coverage": {
    "observed": ["deployment configs"],
    "missing": ["owner mappings"],
    "notes": ["repo evidence preserved"]
  }
}`)

	normalized, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if !changed {
		t.Fatalf("expected legacy manifest repair op to be dropped")
	}

	parsed, err := contracts.ParseTaskResult(normalized)
	if err != nil {
		t.Fatalf("expected normalized taskresult to parse: %v", err)
	}
	if len(parsed.Changeset) != 0 {
		t.Fatalf("expected normalized changeset to be empty, got %#v", parsed.Changeset)
	}
}

func TestNormalizeRawTaskResultKeepsLegacyManifestRepairOpWhenShardManifestIsStillInvalid(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatalf("write invalid shard-pack-manifest: %v", err)
	}

	task := acpruntime.Task{
		StepID:    "init.step1.collect",
		WriteRoot: writeRoot,
	}
	manifestPath := filepath.ToSlash(filepath.Join(writeRoot, "shard-pack-manifest.json"))
	raw := []byte(fmt.Sprintf(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step1.collect",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-18T21:54:16Z"
  },
  "summary": "manifest repaired",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "artifact": {
        "id": "shard-pack-manifest.json",
        "path": %q
      }
    }
  ]
}`, manifestPath))

	_, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if changed {
		t.Fatalf("expected invalid shard manifest to keep legacy repair op for hard failure handling")
	}
}

func TestNormalizeRawTaskResultPreservesValidDocArtifact(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{StepID: "init.step1.collect", WriteRoot: "/tmp/acp/write-root"}
	raw := []byte(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step1.collect",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-18T21:54:16Z"
  },
  "summary": "doc artifact emitted",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "doc_artifact": {
        "id": "doc.1",
        "kind": "analysis",
        "title": "Analysis",
        "path": "reports/as-is/service.md"
      }
    }
  ]
}`)

	normalized, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if changed {
		t.Fatalf("expected valid doc_artifact op to remain untouched")
	}
	if string(normalized) != string(raw) {
		t.Fatalf("expected raw payload to stay unchanged")
	}
}

func TestNormalizeRawTaskResultDoesNotDropNonManifestLegacyArtifact(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{StepID: "init.step1.collect", WriteRoot: "/tmp/acp/write-root"}
	raw := []byte(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step1.collect",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-18T21:54:16Z"
  },
  "summary": "legacy op for real doc",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "artifact": {
        "id": "doc.1",
        "path": "reports/as-is/service.md"
      }
    }
  ]
}`)

	_, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if changed {
		t.Fatalf("expected non-manifest legacy op to remain for hard failure handling")
	}
}

func TestNormalizeRawTaskResultDropsLegacyDraftManifestArtifactForStep2WhenManifestIsValid(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "overview.md"), []byte("# Overview\n"), 0o644); err != nil {
		t.Fatalf("write draft overview: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "overview.md",
      "canonical_path": "reports/as-is/overview.md",
      "kind": "report",
      "title": "Overview"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, "asis-draft-manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write asis draft manifest: %v", err)
	}

	task := acpruntime.Task{
		StepID:         "init.step2.asis_docs",
		WriteRoot:      writeRoot,
		DraftFinalRoot: draftRoot,
		ArtifactRoot:   "reports/taskruns/run-1/runtime/step2_as_is",
	}
	raw := []byte(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step2.asis_docs",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-18T21:54:16Z"
  },
  "summary": "draft manifest already written",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "doc_artifact": {
        "id": "asis-draft-manifest",
        "path": "reports/taskruns/run-1/runtime/step2_as_is/asis-draft-manifest.json"
      }
    }
  ]
}`)

	normalized, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if !changed {
		t.Fatalf("expected legacy step2 draft manifest op to be dropped")
	}
	parsed, err := contracts.ParseTaskResult(normalized)
	if err != nil {
		t.Fatalf("expected normalized step2 taskresult to parse: %v", err)
	}
	if len(parsed.Changeset) != 0 {
		t.Fatalf("expected normalized changeset to be empty, got %#v", parsed.Changeset)
	}
}

func TestNormalizeRawTaskResultKeepsLegacyDraftManifestArtifactWhenReferencedDraftsAreMissing(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "proposal.md",
      "canonical_path": "proposals/proposal-baseline/proposal.md",
      "kind": "proposal",
      "title": "proposal.md"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, "proposals-draft-manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write proposals draft manifest: %v", err)
	}

	task := acpruntime.Task{
		StepID:         "init.step4.proposals",
		WriteRoot:      writeRoot,
		DraftFinalRoot: draftRoot,
		ArtifactRoot:   "reports/taskruns/run-1/runtime/step4_proposals",
	}
	raw := []byte(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step4.proposals",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-18T21:54:16Z"
  },
  "summary": "legacy proposal artifact op remains",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "doc_artifact": {
        "id": "proposal-draft-manifest",
        "path": "reports/taskruns/run-1/runtime/step4_proposals/proposals-draft-manifest.json"
      }
    }
  ]
}`)

	_, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if changed {
		t.Fatalf("expected legacy op to remain when referenced proposal draft files are missing")
	}
}

func TestNormalizeRawTaskResultDropsLegacyDraftManifestArtifactForStep0WhenManifestIsValid(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "charter-overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "baseline-subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write baseline subagents: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "charter-overview.md",
      "canonical_path": "charter/overview.md",
      "kind": "charter",
      "title": "Constitution"
    },
    {
      "path": "baseline-subagents.yaml",
      "canonical_path": "skills/subagents.yaml",
      "kind": "bundle",
      "title": "Baseline Subagents"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, "constitution-draft.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write constitution draft manifest: %v", err)
	}

	task := acpruntime.Task{
		StepID:         "init.step0.constitution",
		WriteRoot:      writeRoot,
		DraftFinalRoot: draftRoot,
		ArtifactRoot:   "reports/taskruns/run-1/runtime/step0_constitution",
	}
	raw := []byte(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step0.constitution",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-20T07:16:00Z"
  },
  "summary": "constitution draft already written",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "doc_artifact": {
        "id": "constitution-draft-manifest",
        "path": "reports/taskruns/run-1/runtime/step0_constitution/constitution-draft.json"
      }
    }
  ]
}`)

	normalized, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if !changed {
		t.Fatalf("expected legacy step0 draft manifest op to be dropped")
	}
	parsed, err := contracts.ParseTaskResult(normalized)
	if err != nil {
		t.Fatalf("expected normalized step0 taskresult to parse: %v", err)
	}
	if len(parsed.Changeset) != 0 {
		t.Fatalf("expected normalized changeset to be empty, got %#v", parsed.Changeset)
	}
}

func TestNormalizeRawTaskResultKeepsLegacyStep0DraftManifestArtifactWhenManifestIsInvalid(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "charter-overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "baseline-subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write baseline subagents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "constitution-draft.json"), []byte(`{"schema_version":"constitution-v0"}`), 0o644); err != nil {
		t.Fatalf("write invalid constitution draft manifest: %v", err)
	}

	task := acpruntime.Task{
		StepID:         "init.step0.constitution",
		WriteRoot:      writeRoot,
		DraftFinalRoot: draftRoot,
		ArtifactRoot:   "reports/taskruns/run-1/runtime/step0_constitution",
	}
	raw := []byte(`{
  "meta": {
    "task_id": "task-1",
    "step_id": "init.step0.constitution",
    "runtime": {"name": "qwen-code"},
    "started_at": "2026-04-20T07:16:00Z"
  },
  "summary": "legacy constitution artifact op remains",
  "changeset": [
    {
      "op": "add_doc_artifact",
      "doc_artifact": {
        "id": "constitution-draft-manifest",
        "path": "reports/taskruns/run-1/runtime/step0_constitution/constitution-draft.json"
      }
    }
  ]
}`)

	_, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil {
		t.Fatalf("normalize raw taskresult: %v", err)
	}
	if changed {
		t.Fatalf("expected legacy step0 op to remain when manifest is invalid")
	}
}
