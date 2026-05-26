package providercommon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
