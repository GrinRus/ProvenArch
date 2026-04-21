package taskresultcompat

import (
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestNormalizeRawTaskResultDropsLegacyManifestRepairOp(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		StepID:    "init.step1.collect",
		WriteRoot: "/tmp/acp/write-root",
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
