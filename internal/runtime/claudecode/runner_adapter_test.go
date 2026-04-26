package claudecode

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestClaudeDefaultArgsKeepNoninteractiveDiagnosticMode(t *testing.T) {
	t.Parallel()

	args := buildClaudeArgsWithIncludeDirectories([]string{"/tmp/workspace", "/tmp/repo"}, "artifact-only prompt")
	assertClaudeArg(t, args, "--output-format")
	assertClaudeArg(t, args, "json")
	assertClaudeArg(t, args, "--permission-mode")
	assertClaudeArg(t, args, "bypassPermissions")
	assertClaudeArg(t, args, "--add-dir")
	assertClaudeArg(t, args, "/tmp/repo")
	assertClaudeArg(t, args, "-p")
	assertClaudeArg(t, args, "artifact-only prompt")
}

func TestClaudeAdapterCommandSpecExposesRuntimeContractSurface(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := filepath.Join(workspace, "repo")
	draftRoot := filepath.Join(workspace, "reports", "taskruns", "run-claude", "staging", "final")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-claude", "runtime", "step0")
	for _, dir := range []string{repoRoot, draftRoot, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	task := acpruntime.Task{
		TaskID:           "task-claude",
		RunID:            "run-claude",
		StepID:           "init.step0.constitution",
		StepContract:     "constitution",
		AgentRole:        "architect",
		Workspace:        workspace,
		WriteRoot:        writeRoot,
		DraftFinalRoot:   draftRoot,
		ReadContextRoots: []string{workspace, repoRoot},
		StartedAtUTC:     time.Now().UTC(),
	}

	spec, err := (claudeAdapter{runner: HeadlessRunner{Command: "claude-test"}}).CommandSpec(task)
	if err != nil {
		t.Fatalf("command spec: %v", err)
	}
	if got, want := spec.Command, "claude-test"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
	if got, want := spec.Dir, draftRoot; got != want {
		t.Fatalf("dir = %q, want %q", got, want)
	}
	if !stringSliceContains(spec.IncludeDirs, repoRoot) {
		t.Fatalf("expected include dirs to expose repo root %q, got %v", repoRoot, spec.IncludeDirs)
	}
	if spec.Stdin == nil {
		t.Fatal("expected JSON task stdin")
	}
	raw, err := io.ReadAll(spec.Stdin)
	if err != nil {
		t.Fatalf("read stdin: %v", err)
	}
	if !strings.Contains(string(raw), "run-claude") {
		t.Fatalf("expected task JSON stdin to contain run id, got %s", raw)
	}
}

func TestClaudeRepairCommandSpecNarrowsReadSurface(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	repoRoot := filepath.Join(root, "repo-a")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "repo-a")
	stagedFinal := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "final")
	for _, dir := range []string{workspace, repoRoot, writeRoot, stagedFinal} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: repo-a\n    path: " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspace,
		WriteRoot:        writeRoot,
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/repo-a",
		ReadContextRoots: []string{workspace, stagedFinal, repoRoot},
		RepoScopes:       []string{"repo-a"},
		PathScopes:       []string{"src"},
	}

	spec, err := (claudeAdapter{runner: HeadlessRunner{Command: "claude-test"}}).CollectManifestRepairCommandSpec(task, os.ErrNotExist)
	if err != nil {
		t.Fatalf("repair command spec: %v", err)
	}
	if stringSliceContains(spec.IncludeDirs, workspace) || stringSliceContains(spec.IncludeDirs, stagedFinal) {
		t.Fatalf("repair include dirs must exclude workspace/taskrun history, got %v", spec.IncludeDirs)
	}
	if !stringSliceContains(spec.IncludeDirs, writeRoot) || !stringSliceContains(spec.IncludeDirs, repoRoot) {
		t.Fatalf("repair include dirs must keep write root and repo evidence, got %v", spec.IncludeDirs)
	}
	raw, err := io.ReadAll(spec.Stdin)
	if err != nil {
		t.Fatalf("read stdin: %v", err)
	}
	if strings.Contains(string(raw), stagedFinal) {
		t.Fatalf("repair task stdin must not expose sibling taskrun roots, got %s", raw)
	}
}

func TestClaudeAdapterUsesSharedUnavailableMarkers(t *testing.T) {
	t.Parallel()

	markers := (claudeAdapter{}).UnavailableMarkers()
	if !stringSliceContains(markers, "rate limit") || !stringSliceContains(markers, "ssl") {
		t.Fatalf("expected shared unavailable markers, got %v", markers)
	}
}

func TestClaudeAdapterMonitorsPreArtifactStallsForArtifactSteps(t *testing.T) {
	t.Parallel()

	policy := (claudeAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected collect artifact and pre-artifact monitoring, got %+v", policy)
	}

	policy = (claudeAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step0.constitution"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected draft artifact and pre-artifact monitoring, got %+v", policy)
	}
}

func assertClaudeArg(t *testing.T, args []string, want string) {
	t.Helper()
	if !stringSliceContains(args, want) {
		t.Fatalf("expected arg %q in %v", want, args)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
