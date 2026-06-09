package codexcode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/testutil"
)

func TestCodexCommandNamePrefersEnvOverride(t *testing.T) {
	t.Setenv("ACP_CODEX_CMD", "/custom/codex")

	runner := HeadlessRunner{}
	if got, want := runner.commandName(), "/custom/codex"; got != want {
		t.Fatalf("expected env override command %q, got %q", want, got)
	}
}

func TestCodexCommandNameDefaultsToCodex(t *testing.T) {
	runner := HeadlessRunner{}
	if got, want := runner.commandName(), "codex"; got != want {
		t.Fatalf("expected default command %q, got %q", want, got)
	}
}

func TestHeadlessRunnerPreflightFailsWhenCommandMissing(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{Command: "definitely-missing-acp-codex-command"}
	err := runner.Preflight(context.Background())
	if err == nil {
		t.Fatalf("expected preflight error")
	}
	if !strings.Contains(err.Error(), "codex-code") {
		t.Fatalf("expected codex-code in error, got %v", err)
	}
}

func TestDefaultCodexArgsKeepNoninteractiveDiagnosticMode(t *testing.T) {
	t.Parallel()

	args := buildCodexArgsWithIncludeDirectories("/tmp/work", []string{"/tmp/work", "/tmp/repo"})
	assertCodexArg(t, args, "exec")
	assertCodexArg(t, args, "--json")
	assertCodexArg(t, args, "--color")
	assertCodexArg(t, args, "never")
	assertCodexArg(t, args, "--skip-git-repo-check")
	assertCodexArg(t, args, "--sandbox")
	assertCodexArg(t, args, "danger-full-access")
	assertCodexArg(t, args, "--cd")
	assertCodexArg(t, args, "/tmp/work")
	assertCodexArg(t, args, "--add-dir")
	assertCodexArg(t, args, "/tmp/repo")
	assertCodexArg(t, args, "--ephemeral")
	assertCodexArg(t, args, "-")
}

func TestManagedCodexArgsOmitDangerFullAccess(t *testing.T) {
	t.Parallel()

	args := buildCodexArgsWithPermissions("/tmp/work", []string{"/tmp/work", "/tmp/repo"}, acpruntime.PermissionValues{Mode: acpruntime.PermissionModeManaged})
	if codexSliceContains(args, "danger-full-access") {
		t.Fatalf("managed mode must omit danger-full-access, got %v", args)
	}
	assertCodexArg(t, args, "--sandbox")
	assertCodexArg(t, args, "workspace-write")
}

func TestCodexAdapterUsesSharedUnavailableMarkers(t *testing.T) {
	t.Parallel()

	markers := (codexAdapter{}).UnavailableMarkers()
	if !codexSliceContains(markers, "rate limit") || !codexSliceContains(markers, "ssl") {
		t.Fatalf("expected shared unavailable markers, got %v", markers)
	}
}

func TestCodexAdapterMonitorsPreArtifactStallsForArtifactSteps(t *testing.T) {
	t.Parallel()

	policy := (codexAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected collect artifact and pre-artifact monitoring, got %+v", policy)
	}
	if policy.PreArtifactStallWindow != 0 {
		t.Fatalf("codex must keep default shared pre-artifact window, got %+v", policy)
	}
	if got, want := policy.PostArtifactStallWindow, 90*time.Second; got != want {
		t.Fatalf("expected codex collect post-artifact enrichment window %s, got %s", want, got)
	}
	if got, want := policy.PartialArtifactStallWindow, 90*time.Second; got != want {
		t.Fatalf("expected codex collect partial-artifact enrichment window %s, got %s", want, got)
	}

	policy = (codexAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step0.constitution"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected draft artifact and pre-artifact monitoring, got %+v", policy)
	}
	if policy.PreArtifactStallWindow != 0 {
		t.Fatalf("codex draft policy must keep default shared pre-artifact window, got %+v", policy)
	}
	if policy.PostArtifactStallWindow != 0 || policy.PartialArtifactStallWindow != 0 {
		t.Fatalf("draft steps must keep shared post-artifact defaults, got %+v", policy)
	}
}

func TestHeadlessRunnerSucceedsWithValidDraftArtifacts(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	writeRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "step0")
	draftRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	runner := newStubHeadlessRunner(writeDraftStubRunner(t), writeRoot, draftRoot, "run-codex")
	task := acpruntime.Task{
		RunID:             "run-codex",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		Workspace:         workspaceDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
		StartedAtUTC:      time.Now().UTC(),
	}

	result, err := runner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("run codex draft runner: %v", err)
	}
	if result.Execution.Provider != string(acpruntime.ProviderCodexCode) {
		t.Fatalf("expected codex provider in execution, got %+v", result.Execution)
	}
	if _, err := os.Stat(filepath.Join(writeRoot, "constitution-draft.json")); err != nil {
		t.Fatalf("expected constitution-draft.json: %v", err)
	}
}

func TestHeadlessRunnerFailsWhenArtifactsAreMissing(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	writeRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "step0")
	draftRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	runner := newStubHeadlessRunner(writeNoopStubRunner(t), "noop")
	task := acpruntime.Task{
		RunID:             "run-codex",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		Workspace:         workspaceDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
		StartedAtUTC:      time.Now().UTC(),
	}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatalf("expected runtime contract error")
	}
	if !strings.Contains(err.Error(), "contract") {
		t.Fatalf("expected contract failure, got %v", err)
	}
}

func TestHeadlessRunnerPreservesStdoutExcerptOnProcessFailure(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	writeRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "step0")
	draftRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	runner := newStubHeadlessRunner(writeFailingStubRunner(t), "noop")
	task := acpruntime.Task{
		RunID:             "run-codex",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		Workspace:         workspaceDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
		StartedAtUTC:      time.Now().UTC(),
	}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatalf("expected execution failure")
	}
	if !strings.Contains(err.Error(), "stdout_excerpt=") {
		t.Fatalf("expected stdout excerpt in failure, got %v", err)
	}
}

func TestHeadlessRunnerClassifiesTimeoutAndWritesDiagnostics(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	writeRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "step0")
	draftRoot := filepath.Join(workspaceDir, "reports", "taskruns", "run-codex", "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	runner := HeadlessRunner{
		Command: "bash",
		Args:    []string{"-c", "printf '%s\n' 'codex stub started'; sleep 5"},
	}
	task := acpruntime.Task{
		TaskID:            "task-timeout",
		RunID:             "run-codex",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		Workspace:         workspaceDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
		StartedAtUTC:      time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := runner.Run(ctx, task)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runtime runner error, got %T: %v", err, err)
	}
	if got, want := runnerErr.Code, acpruntime.ErrorCodeRuntimeTimeout; got != want {
		t.Fatalf("expected %s, got %s (%v)", want, got, err)
	}
	if strings.TrimSpace(runnerErr.RawOutputRefs.Metadata) == "" {
		t.Fatalf("expected timeout raw metadata refs")
	}

	rawMeta, readErr := os.ReadFile(filepath.Join(workspaceDir, filepath.FromSlash(runnerErr.RawOutputRefs.Metadata)))
	if readErr != nil {
		t.Fatalf("read timeout metadata: %v", readErr)
	}
	meta := map[string]any{}
	if decodeErr := json.Unmarshal(rawMeta, &meta); decodeErr != nil {
		t.Fatalf("decode timeout metadata: %v", decodeErr)
	}
	diagnostics, ok := meta["diagnostics"].(map[string]any)
	if !ok {
		t.Fatalf("expected diagnostics block in timeout metadata: %#v", meta)
	}
	if got := strings.TrimSpace(diagnostics["current_step"].(string)); got != "init.step0.constitution" {
		t.Fatalf("unexpected current_step %q", got)
	}
	if _, ok := diagnostics["last_stdout_bytes"]; !ok {
		t.Fatalf("expected last_stdout_bytes in timeout diagnostics: %#v", diagnostics)
	}
}

func writeDraftStubRunner(t *testing.T) string {
	t.Helper()

	script := `#!/usr/bin/env bash
set -eu
write_root="$1"
draft_root="$2"
run_id="$3"
mkdir -p "$write_root" "$draft_root"
cat >"$write_root/constitution-draft.json" <<EOF
{
  "version": 1,
  "run_id": "$run_id",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "summary": "Codex draft stub",
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
}
EOF
cat >"$draft_root/charter-overview.md" <<'EOF'
# Constitution
EOF
cat >"$draft_root/baseline-subagents.yaml" <<'EOF'
version: 1
EOF
printf '%s\n' '{"type":"result","status":"ok"}'
`
	return testutil.WriteExecutableScript(t, "codex-draft-stub.sh", script)
}

func writeNoopStubRunner(t *testing.T) string {
	t.Helper()

	script := "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' '{\"type\":\"result\",\"status\":\"ok\"}'\n"
	return testutil.WriteExecutableScript(t, "codex-noop-stub.sh", script)
}

func writeFailingStubRunner(t *testing.T) string {
	t.Helper()

	script := "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' 'codex stub failed after emitting output'\nexit 1\n"
	return testutil.WriteExecutableScript(t, "codex-failing-stub.sh", script)
}

func newStubHeadlessRunner(script string, args ...string) HeadlessRunner {
	return HeadlessRunner{
		Command: "bash",
		Args:    append([]string{script}, args...),
	}
}

func assertCodexArg(t *testing.T, args []string, want string) {
	t.Helper()
	if !codexSliceContains(args, want) {
		t.Fatalf("expected arg %q in %v", want, args)
	}
}

func codexSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
