package artifactquality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestEnsureCanonicalCollectManifestNormalizesArtifactRootRelativeDocumentPaths(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run_20260420_054749_001", "staging", "shards", "bank-of-anthos-iac")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write_root: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "bank_of_anthos_doubled_path_manifest.json"))
	if err != nil {
		t.Fatalf("read fixture manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "iac-overview.md"), []byte("# IAC Overview\n"), 0o644); err != nil {
		t.Fatalf("seed authored doc: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run_20260420_054749_001",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run_20260420_054749_001/staging/shards/bank-of-anthos-iac",
		StartedAtUTC: time.Date(2026, 4, 20, 5, 47, 49, 0, time.UTC),
	}
	result := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    "task-1",
			StepID:    "init.step1.collect",
			Runtime:   contracts.RuntimeMeta{Name: "qwen-code", Version: "test"},
			StartedAt: "2026-04-20T05:47:49Z",
		},
		Summary:   "Collected infrastructure shard.",
		Changeset: []contracts.Operation{},
	}

	if err := EnsureCanonicalCollectManifest(task, result); err != nil {
		t.Fatalf("ensure canonical collect manifest: %v", err)
	}

	normalizedRaw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read normalized manifest: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(normalizedRaw)
	if err != nil {
		t.Fatalf("parse normalized manifest: %v", err)
	}
	if got := manifest.Documents[0].Path; got != "iac-overview.md" {
		t.Fatalf("expected artifact_root-relative document path, got %q", got)
	}
	if got := manifest.Documents[0].CanonicalPath; got != "reports/as-is/bank-of-anthos-iac-overview.md" {
		t.Fatalf("unexpected canonical path %q", got)
	}
	if got := manifest.Documents[0].CitationIDs; len(got) != 1 || got[0] != "cite.bank.readme" {
		t.Fatalf("unexpected citation ids %#v", got)
	}
}

func TestCanonicalizeCollectManifestDoesNotRewriteUnknownWorkspaceRelativeDocumentPath(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "documents": [
    {
      "id": "doc.iac-overview",
      "kind": "analysis",
      "title": "IAC Overview",
      "path": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac/missing.md",
      "canonical_path": "reports/as-is/bank-of-anthos-iac-overview.md",
      "topics": ["iac"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "iac/README.md",
      "claim_ids": ["claim.iac.readme"],
      "document_ids": ["doc.iac-overview"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["iac"], "missing": ["owner mappings"], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)

	_, _, err := canonicalizeCollectManifest(raw, acpruntime.Task{
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
	}, contracts.TaskResult{})
	if err == nil {
		t.Fatalf("expected invalid manifest to stay invalid when staged file is missing")
	}
}

func TestNormalizeCollectDocumentPathNormalizesAbsolutePathInsideWriteRoot(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	absDocPath := filepath.Join(writeRoot, "nested", "report.md")
	if err := os.MkdirAll(filepath.Dir(absDocPath), 0o755); err != nil {
		t.Fatalf("mkdir doc dir: %v", err)
	}
	if err := os.WriteFile(absDocPath, []byte("# Report\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	relPath, ok := normalizeCollectDocumentPath(absDocPath, "reports/taskruns/run-1/staging/shards/sample", writeRoot)
	if !ok {
		t.Fatalf("expected absolute path to normalize")
	}
	if relPath != "nested/report.md" {
		t.Fatalf("unexpected normalized path %q", relPath)
	}
}

func TestEnsureCanonicalCollectManifestPreservesCompatibilityProjection(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "sample")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write_root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "report.md"), []byte("# Report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "sample",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/sample",
  "documents": [
    {
      "id": "doc.sample",
      "kind": "analysis",
      "title": "Sample",
      "path": "report.md",
      "canonical_path": "reports/as-is/sample/report.md",
      "topics": ["sample"],
      "citation_ids": ["cite.sample"]
    }
  ],
  "citations": [
    {
      "id": "cite.sample",
      "repo": "sample",
      "path": "README.md",
      "claim_ids": ["claim.sample"],
      "document_ids": ["doc.sample"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	result := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    "task-compat",
			StepID:    "init.step1.collect",
			Runtime:   contracts.RuntimeMeta{Name: "qwen-code", Version: "test"},
			StartedAt: "2026-04-20T12:00:00Z",
		},
		Summary: "ok",
		Coverage: &contracts.Coverage{
			Observed: []string{"sample"},
			Missing:  []string{"owners"},
			Notes:    []string{"compat propagated"},
		},
		Changeset: []contracts.Operation{},
	}
	if err := EnsureCanonicalCollectManifest(acpruntime.Task{
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/sample",
	}, result); err != nil {
		t.Fatalf("ensure canonical manifest: %v", err)
	}

	var manifest contracts.ShardPackManifest
	normalizedRaw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(normalizedRaw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if got := manifest.Compatibility.Coverage.Observed; len(got) != 1 || got[0] != "sample" {
		t.Fatalf("unexpected compatibility coverage %#v", got)
	}
}

func TestEnsureCanonicalCollectManifestPreservesRichManifestCompatibilityWhenRetryTaskResultIsMinimal(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "sample")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write_root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "report.md"), []byte("# Report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "sample",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/sample",
  "documents": [
    {
      "id": "doc.sample",
      "kind": "analysis",
      "title": "Sample",
      "path": "report.md",
      "canonical_path": "reports/as-is/sample/report.md",
      "topics": ["sample"],
      "citation_ids": ["cite.sample"]
    }
  ],
  "citations": [
    {
      "id": "cite.sample",
      "repo": "sample",
      "path": "README.md",
      "claim_ids": ["claim.sample"],
      "document_ids": ["doc.sample"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["sample"], "missing": ["owners"], "notes": ["rich manifest"]},
    "questions": [{"id": "q.sample.owner", "text": "Who owns sample?", "priority": "high"}],
    "entities": [{"id": "svc.sample", "type": "service", "name": "Sample Service", "provenance": {"kind": "observation", "confidence": 0.8, "evidence": [{"repo": "sample", "path": "README.md"}]}}],
    "edges": [{"id": "edge.sample.db", "type": "depends_on", "from": "svc.sample", "to": "db.sample", "provenance": {"kind": "inference", "confidence": 0.6, "evidence": [{"repo": "sample", "path": "README.md"}]}}],
    "findings": [{"id": "finding.sample", "title": "Sample finding", "description": "Sample finding detail", "severity": "medium", "provenance": {"kind": "observation", "confidence": 0.7, "evidence": [{"repo": "sample", "path": "README.md"}]}}]
  }
}`)
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	result := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    "task-minimal-retry",
			StepID:    "init.step1.collect",
			Runtime:   contracts.RuntimeMeta{Name: "qwen-code", Version: "test"},
			StartedAt: "2026-04-20T12:00:00Z",
		},
		Summary:   "minimal retry reused authored docs",
		Changeset: []contracts.Operation{},
	}
	if err := EnsureCanonicalCollectManifest(acpruntime.Task{
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/sample",
	}, result); err != nil {
		t.Fatalf("ensure canonical manifest: %v", err)
	}

	normalizedRaw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(normalizedRaw)
	if err != nil {
		t.Fatalf("parse normalized manifest: %v", err)
	}
	if got := manifest.Compatibility.Coverage.Observed; len(got) != 1 || got[0] != "sample" {
		t.Fatalf("unexpected coverage %#v", got)
	}
	if len(manifest.Compatibility.Questions) != 1 || manifest.Compatibility.Questions[0].ID != "q.sample.owner" {
		t.Fatalf("unexpected questions %#v", manifest.Compatibility.Questions)
	}
	if len(manifest.Compatibility.Entities) != 1 || manifest.Compatibility.Entities[0].ID != "svc.sample" {
		t.Fatalf("unexpected entities %#v", manifest.Compatibility.Entities)
	}
	if len(manifest.Compatibility.Edges) != 1 || manifest.Compatibility.Edges[0].ID != "edge.sample.db" {
		t.Fatalf("unexpected edges %#v", manifest.Compatibility.Edges)
	}
	if len(manifest.Compatibility.Findings) != 1 || manifest.Compatibility.Findings[0].ID != "finding.sample" {
		t.Fatalf("unexpected findings %#v", manifest.Compatibility.Findings)
	}
}
