package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestFakeRunnerWritesShardPackManifestWhenWriteRootProvided(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "payments")
	runner := FakeRunner{}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-collect",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		ShardID:      "payments",
		DomainID:     "payments",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/payments",
		AgentRole:    "shard-analyst",
		RepoScopes:   []string{"payments-service"},
		PathScopes:   []string{"."},
		StartedAtUTC: time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run fake collect: %v", err)
	}
	if len(result.TaskResult.Changeset) == 0 {
		t.Fatalf("expected collect changeset")
	}

	manifestRaw, err := os.ReadFile(filepath.Join(writeRoot, "shard-pack-manifest.json"))
	if err != nil {
		t.Fatalf("read shard-pack-manifest.json: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(manifestRaw)
	if err != nil {
		t.Fatalf("parse shard-pack-manifest.json: %v", err)
	}
	if len(manifest.Documents) == 0 {
		t.Fatalf("expected authored documents in manifest")
	}
	if len(manifest.Citations) == 0 {
		t.Fatalf("expected citations in manifest")
	}
}

func TestFakeRunnerWritesValidatorVerdictWhenWriteRootProvided(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	stageRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "final")
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		t.Fatalf("mkdir stage root: %v", err)
	}
	for _, name := range []string{"final-run-index.json", "citation-index.json"} {
		if err := os.WriteFile(filepath.Join(stageRoot, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "validator")
	runner := FakeRunner{}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-findings",
		RunID:        "run-1",
		StepID:       "init.step3.findings",
		ShardID:      "validator",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/validator",
		AgentRole:    "validator",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run fake findings: %v", err)
	}
	if len(result.TaskResult.Changeset) == 0 {
		t.Fatalf("expected findings changeset")
	}

	verdictRaw, err := os.ReadFile(filepath.Join(writeRoot, "validator-verdict.json"))
	if err != nil {
		t.Fatalf("read validator-verdict.json: %v", err)
	}
	verdict, err := contracts.ParseValidatorVerdict(verdictRaw)
	if err != nil {
		t.Fatalf("parse validator-verdict.json: %v", err)
	}
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected PASS verdict, got %q", verdict.Verdict)
	}
	if len(verdict.CheckedPaths) == 0 {
		t.Fatalf("expected checked paths in verdict")
	}
}
