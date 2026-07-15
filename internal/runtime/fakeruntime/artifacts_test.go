package fakeruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestAsIsDraftDoesNotLeakRunLocalStagingPaths(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	draftRoot := filepath.Join(workspace, "reports", "taskruns", "run-demo", "staging", "final")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-demo", "runtime", "step2_as_is")
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	task := acpruntime.Task{
		TaskID: "task-as-is", RunID: "run-demo", StepID: "init.step2.asis_docs", StepContract: "as_is", AgentRole: "architect",
		Workspace: workspace, WriteRoot: writeRoot, DraftFinalRoot: draftRoot,
		RepoScopes: []string{"payments-service"}, StartedAtUTC: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
	}
	if _, err := (Runner{}).Run(context.Background(), task); err != nil {
		t.Fatalf("run fake as-is: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(draftRoot, "overview.md"))
	if err != nil {
		t.Fatalf("read overview: %v", err)
	}
	if strings.Contains(string(raw), "reports/taskruns/") || strings.Contains(string(raw), "/staging/") {
		t.Fatalf("fake user-visible evidence leaked staging path:\n%s", raw)
	}
	for _, heading := range []string{"System at a glance", "Analyzed scope", "Domains and ownership", "Key flows", "Integrations and datastores", "Where to start", "Safe-change guidance", "Evidence gaps and open questions"} {
		if !strings.Contains(string(raw), "## "+heading) {
			t.Fatalf("fake Architecture Home is missing %q:\n%s", heading, raw)
		}
	}
}

func TestRunnerWritesShardPackManifestWhenWriteRootProvided(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "payments")
	runner := Runner{}

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
	if result.Execution.TaskID != "task-collect" {
		t.Fatalf("expected runtime execution metadata for collect task")
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
	if len(manifest.Semantic.Entities) == 0 {
		t.Fatalf("expected semantic entities in manifest")
	}
}

func TestRunnerUsesExistingRepoEvidencePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoRoot := filepath.Join(workspace, ".acp", "repos", "ftgo-application-d542d7e34d40")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.adoc"), []byte("= FTGO\n"), 0o644); err != nil {
		t.Fatalf("write README.adoc: %v", err)
	}
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "ftgo-root")
	runner := Runner{}

	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:           "task-collect",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		ShardID:          "ftgo-root",
		DomainID:         "ftgo-root",
		Workspace:        workspace,
		WriteRoot:        writeRoot,
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/ftgo-root",
		AgentRole:        "shard-analyst",
		RepoScopes:       []string{"ftgo-application"},
		PathScopes:       []string{"."},
		ReadContextRoots: []string{workspace, repoRoot},
		StartedAtUTC:     time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run fake collect: %v", err)
	}

	manifestRaw, err := os.ReadFile(filepath.Join(writeRoot, "shard-pack-manifest.json"))
	if err != nil {
		t.Fatalf("read shard-pack-manifest.json: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(manifestRaw)
	if err != nil {
		t.Fatalf("parse shard-pack-manifest.json: %v", err)
	}
	if got, want := manifest.Citations[0].Path, "README.adoc"; got != want {
		t.Fatalf("expected fake citation to use existing evidence path %q, got %q", want, got)
	}
	if got, want := manifest.Semantic.Entities[0].Provenance.Evidence[0].Path, "README.adoc"; got != want {
		t.Fatalf("expected fake semantic evidence to use existing path %q, got %q", want, got)
	}
}

func TestRunnerWritesValidatorVerdictWhenWriteRootProvided(t *testing.T) {
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
	runner := Runner{}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-findings",
		RunID:        "run-1",
		StepID:       "init.step3.findings",
		ShardID:      "validator",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/validator",
		AgentRole:    "validator-findings",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run fake findings: %v", err)
	}
	if result.Execution.TaskID != "task-findings" {
		t.Fatalf("expected runtime execution metadata for findings task")
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
	if len(verdict.Findings) == 0 {
		t.Fatalf("expected findings in verdict")
	}
}
