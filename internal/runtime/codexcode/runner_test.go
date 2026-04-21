package codexcode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

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

	runner := HeadlessRunner{
		Command: writeDraftStubRunner(t),
		Args:    []string{writeRoot, draftRoot, "run-codex"},
	}
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

	runner := HeadlessRunner{
		Command: writeNoopStubRunner(t),
		Args:    []string{"noop"},
	}
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

	runner := HeadlessRunner{
		Command: writeFailingStubRunner(t),
		Args:    []string{"noop"},
	}
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

func writeDraftStubRunner(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "codex-draft-stub.sh")
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
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write draft stub: %v", err)
	}
	return path
}

func writeNoopStubRunner(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "codex-noop-stub.sh")
	script := "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' '{\"type\":\"result\",\"status\":\"ok\"}'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write noop stub: %v", err)
	}
	return path
}

func writeFailingStubRunner(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "codex-failing-stub.sh")
	script := "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' 'codex stub failed after emitting output'\nexit 1\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write failing stub: %v", err)
	}
	return path
}
