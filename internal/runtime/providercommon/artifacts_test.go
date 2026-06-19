package providercommon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestValidateQAArtifactsRequiresCitationsFromContextPack(t *testing.T) {
	root := t.TempDir()
	writeRoot := filepath.Join(root, "reports", "taskruns", "run-qa-1", "qa")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	contextPackPath := filepath.Join(writeRoot, "context-pack.json")
	writeJSONFile(t, contextPackPath, map[string]any{
		"version":      1,
		"run_id":       "run-qa-1",
		"question":     "Which overview exists?",
		"generated_at": "2026-05-26T18:30:00Z",
		"documents": []map[string]any{
			{
				"path":    "reports/as-is/overview.md",
				"weight":  6,
				"content": "Architecture overview.",
			},
		},
	})

	task := acpruntime.Task{
		RunID:           "run-qa-1",
		StepID:          acpruntime.StepIDQAAsk,
		WriteRoot:       writeRoot,
		ContextPackPath: contextPackPath,
		Question:        "Which overview exists?",
	}

	writeQAAnswer(t, writeRoot, []map[string]string{{
		"path":   "reports/as-is/overview.md",
		"reason": "selected from context pack",
	}})
	if err := ValidateQAArtifacts(task); err != nil {
		t.Fatalf("expected context-pack citation to validate: %v", err)
	}

	writeJSONFile(t, contextPackPath, map[string]any{
		"version":      1,
		"run_id":       "run-qa-other",
		"question":     "Which overview exists?",
		"generated_at": "2026-05-26T18:30:00Z",
		"documents": []map[string]any{
			{
				"path":    "reports/as-is/overview.md",
				"weight":  6,
				"content": "Architecture overview.",
			},
		},
	})
	err := ValidateQAArtifacts(task)
	if err == nil || !strings.Contains(err.Error(), "does not match context pack run_id") {
		t.Fatalf("expected mismatched context pack run_id to fail, got %v", err)
	}

	writeJSONFile(t, contextPackPath, map[string]any{
		"version":      1,
		"run_id":       "run-qa-1",
		"question":     "Which overview exists?",
		"generated_at": "2026-05-26T18:30:00Z",
		"documents": []map[string]any{
			{
				"path":    "reports/as-is/overview.md",
				"weight":  6,
				"content": "Architecture overview.",
			},
		},
	})
	writeQAAnswer(t, writeRoot, []map[string]string{{
		"path":   "reports/taskruns/run-qa-1/qa/context-pack.json",
		"reason": "audit artifact is not evidence",
	}})
	err = ValidateQAArtifacts(task)
	if err == nil || !strings.Contains(err.Error(), "not present in context pack") {
		t.Fatalf("expected citation outside context pack to fail, got %v", err)
	}
}

func TestValidateQAArtifactsAllowsEmptyCitationsWhenContextHasNoDocuments(t *testing.T) {
	root := t.TempDir()
	writeRoot := filepath.Join(root, "reports", "taskruns", "run-qa-1", "qa")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	contextPackPath := filepath.Join(writeRoot, "context-pack.json")
	writeJSONFile(t, contextPackPath, map[string]any{
		"version":      1,
		"run_id":       "run-qa-1",
		"question":     "Which overview exists?",
		"generated_at": "2026-05-26T18:30:00Z",
		"documents":    []map[string]any{},
	})
	writeQAAnswer(t, writeRoot, []map[string]string{})

	err := ValidateQAArtifacts(acpruntime.Task{
		RunID:           "run-qa-1",
		StepID:          acpruntime.StepIDQAAsk,
		WriteRoot:       writeRoot,
		ContextPackPath: contextPackPath,
		Question:        "Which overview exists?",
	})
	if err != nil {
		t.Fatalf("expected empty citations to validate with empty context pack: %v", err)
	}
}

func TestRuntimeArtifactSnapshotRequiresFullQAContextValidation(t *testing.T) {
	root := t.TempDir()
	writeRoot := filepath.Join(root, "reports", "taskruns", "run-qa-1", "qa")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	contextPackPath := filepath.Join(writeRoot, "context-pack.json")
	writeJSONFile(t, contextPackPath, map[string]any{
		"version":      1,
		"run_id":       "run-qa-1",
		"question":     "Which overview exists?",
		"generated_at": "2026-05-26T18:30:00Z",
		"documents": []map[string]any{
			{
				"path":    "reports/as-is/overview.md",
				"weight":  6,
				"content": "Architecture overview.",
			},
		},
	})
	task := acpruntime.Task{
		RunID:           "run-qa-1",
		StepID:          acpruntime.StepIDQAAsk,
		WriteRoot:       writeRoot,
		ContextPackPath: contextPackPath,
		Question:        "Which overview exists?",
	}

	writeQAAnswer(t, writeRoot, []map[string]string{{
		"path":   "reports/taskruns/run-qa-1/qa/context-pack.json",
		"reason": "audit artifact is not evidence",
	}})
	snapshot := runtimeArtifactSnapshot(task)
	if !snapshot.ArtifactObserved || snapshot.Valid || snapshot.State != "invalid" {
		t.Fatalf("expected invalid observed QA snapshot, got %+v", snapshot)
	}

	writeQAAnswer(t, writeRoot, []map[string]string{{
		"path":   "reports/as-is/overview.md",
		"reason": "selected from context pack",
	}})
	snapshot = runtimeArtifactSnapshot(task)
	if !snapshot.ArtifactObserved || !snapshot.Valid || snapshot.State != "valid" {
		t.Fatalf("expected valid observed QA snapshot, got %+v", snapshot)
	}
}

func TestRuntimeArtifactSnapshotRejectsBootstrapOnlyCollectDocument(t *testing.T) {
	root := t.TempDir()
	writeRoot := filepath.Join(root, "reports", "taskruns", "run-collect-1", "staging", "shards", "payments-src")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	writeCollectManifest(t, writeRoot)
	writeTextFile(t, filepath.Join(writeRoot, "payments-overview.md"), "# Payments Overview\n\n<!-- "+artifactquality.CollectBootstrapReplaceMarker+" -->\n\n## Observations\n- Repository scope: payments.\n")

	snapshot := runtimeArtifactSnapshot(acpruntime.Task{
		RunID:     "run-collect-1",
		StepID:    "init.step1.collect",
		WriteRoot: writeRoot,
	})
	if !snapshot.ArtifactObserved || snapshot.Valid || snapshot.State != "invalid" {
		t.Fatalf("expected bootstrap-only collect snapshot to be invalid, got %+v", snapshot)
	}

	writeTextFile(t, filepath.Join(writeRoot, "payments-overview.md"), "# Payments Overview\n\n## Scope\n- Repository scope: payments.\n- Assigned scope summary: `src`.\n\n## Evidence Summary\n- Primary scoped evidence path: `src`.\n- This initial collect pair is a seed-only scoped evidence surface for the assigned shard.\n\n## Evidence Surface\n- `src`: scoped repository evidence available to this collect shard.\n\n## Initial Findings\n- The assigned evidence surface is traceable, but ownership, runtime responsibility, and escalation details need confirmation from richer repository evidence.\n\n## Coverage Gaps\n- Confirm concrete owners, runtime responsibilities, dependencies, and operational escalation paths for this shard.\n")
	snapshot = runtimeArtifactSnapshot(acpruntime.Task{
		RunID:     "run-collect-1",
		StepID:    "init.step1.collect",
		WriteRoot: writeRoot,
	})
	if !snapshot.ArtifactObserved || snapshot.Valid || snapshot.State != "invalid" {
		t.Fatalf("expected marker-free seed collect snapshot to be invalid, got %+v", snapshot)
	}

	writeTextFile(t, filepath.Join(writeRoot, "payments-overview.md"), "# Payments Overview\n\n## Observations\n- `src/payment_handler.go` defines the payment API.\n\n## Evidence\n- `src/payment_handler.go`\n")
	snapshot = runtimeArtifactSnapshot(acpruntime.Task{
		RunID:     "run-collect-1",
		StepID:    "init.step1.collect",
		WriteRoot: writeRoot,
	})
	if !snapshot.ArtifactObserved || !snapshot.Valid || snapshot.State != "valid" {
		t.Fatalf("expected enriched collect snapshot to be valid, got %+v", snapshot)
	}
}

func TestValidateCollectArtifactsRejectsMissingRepoEvidencePath(t *testing.T) {
	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "workspace")
	repoRoot := filepath.Join(workspaceRoot, ".acp", "repos", "payments-d542d7e34d40")
	writeRoot := filepath.Join(workspaceRoot, "reports", "taskruns", "run-collect-1", "staging", "shards", "payments-src")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	writeCollectManifest(t, writeRoot)
	writeTextFile(t, filepath.Join(writeRoot, "payments-overview.md"), "# Payments Overview\n\n## Observations\n- `src/payment_handler.go` defines the payment API.\n")

	task := acpruntime.Task{
		RunID:            "run-collect-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceRoot,
		WriteRoot:        writeRoot,
		ReadContextRoots: []string{workspaceRoot, repoRoot},
		RepoScope:        "payments",
		RepoScopes:       []string{"payments"},
	}
	err := ValidateCollectArtifacts(task, acpruntime.ProviderCodexCode)
	if err == nil {
		t.Fatalf("expected missing repo evidence path to fail validation")
	}
	if !strings.Contains(err.Error(), `repo evidence path "src/payment_handler.go" is missing`) {
		t.Fatalf("expected missing repo evidence path error, got %v", err)
	}

	writeTextFile(t, filepath.Join(repoRoot, "src", "payment_handler.go"), "package payments\n")
	if err := ValidateCollectArtifacts(task, acpruntime.ProviderCodexCode); err != nil {
		t.Fatalf("expected collect artifacts to validate after repo evidence exists: %v", err)
	}
}

func TestRuntimeArtifactSnapshotRejectsDirectoryRepoEvidencePath(t *testing.T) {
	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "workspace")
	repoRoot := filepath.Join(workspaceRoot, ".acp", "repos", "payments-d542d7e34d40")
	writeRoot := filepath.Join(workspaceRoot, "reports", "taskruns", "run-collect-1", "staging", "shards", "payments-src")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "src", "payment_handler.go"), 0o755); err != nil {
		t.Fatalf("mkdir directory evidence path: %v", err)
	}
	writeCollectManifest(t, writeRoot)
	writeTextFile(t, filepath.Join(writeRoot, "payments-overview.md"), "# Payments Overview\n\n## Observations\n- `src/payment_handler.go` defines the payment API.\n")

	task := acpruntime.Task{
		RunID:            "run-collect-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceRoot,
		WriteRoot:        writeRoot,
		ReadContextRoots: []string{workspaceRoot, repoRoot},
		RepoScope:        "payments",
		RepoScopes:       []string{"payments"},
	}
	snapshot := runtimeArtifactSnapshot(task)
	if !snapshot.ArtifactObserved || snapshot.Valid || snapshot.State != "invalid" {
		t.Fatalf("expected directory evidence path to keep collect snapshot invalid, got %+v", snapshot)
	}

	if err := os.RemoveAll(filepath.Join(repoRoot, "src", "payment_handler.go")); err != nil {
		t.Fatalf("remove directory evidence path: %v", err)
	}
	writeTextFile(t, filepath.Join(repoRoot, "src", "payment_handler.go"), "package payments\n")
	snapshot = runtimeArtifactSnapshot(task)
	if !snapshot.ArtifactObserved || !snapshot.Valid || snapshot.State != "valid" {
		t.Fatalf("expected file evidence path to validate collect snapshot, got %+v", snapshot)
	}
}

func writeQAAnswer(t *testing.T, writeRoot string, citations []map[string]string) {
	t.Helper()
	writeJSONFile(t, filepath.Join(writeRoot, "qa-answer.json"), map[string]any{
		"version":      1,
		"run_id":       "run-qa-1",
		"question":     "Which overview exists?",
		"answer":       "The overview is available from the provided context.",
		"citations":    citations,
		"unresolved":   []string{},
		"confidence":   0.75,
		"provider":     "test-provider",
		"generated_at": "2026-05-26T18:30:01Z",
	})
}

func writeCollectManifest(t *testing.T, writeRoot string) {
	t.Helper()
	writeJSONFile(t, filepath.Join(writeRoot, ShardPackManifestFileName), map[string]any{
		"version":       1,
		"run_id":        "run-collect-1",
		"step_id":       "init.step1.collect",
		"shard_id":      "payments-src",
		"domain_id":     "payments",
		"agent_role":    "shard-analyst",
		"artifact_root": "reports/taskruns/run-collect-1/staging/shards/payments-src",
		"repo_scopes":   []string{"payments"},
		"path_scopes":   []string{"src"},
		"documents": []map[string]any{
			{
				"id":             "doc.payments.src.overview",
				"kind":           "report",
				"title":          "Payments Overview",
				"path":           "payments-overview.md",
				"canonical_path": "reports/as-is/payments-src/payments-overview.md",
				"topics":         []string{"payments"},
				"citation_ids":   []string{"cite.payments.src.overview"},
			},
		},
		"citations": []map[string]any{
			{
				"id":           "cite.payments.src.overview",
				"repo":         "payments",
				"path":         "src/payment_handler.go",
				"claim_ids":    []string{"claim.payments.src.overview"},
				"document_ids": []string{"doc.payments.src.overview"},
			},
		},
		"semantic": map[string]any{
			"coverage": map[string]any{
				"observed": []string{"payments API"},
				"missing":  []string{},
				"notes":    []string{"Observed from concrete source files."},
			},
			"questions": []map[string]any{},
			"entities": []map[string]any{
				{
					"id":   "svc.payments",
					"type": "service",
					"name": "Payments",
					"provenance": map[string]any{
						"kind":       "observation",
						"confidence": 0.8,
						"evidence": []map[string]any{
							{"repo": "payments", "path": "src/payment_handler.go"},
						},
					},
				},
			},
			"edges":    []map[string]any{},
			"findings": []map[string]any{},
		},
	})
}

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeTextFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
