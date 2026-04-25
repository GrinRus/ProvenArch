package qwencode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRunClassifiesProviderUnavailableWhenArtifactsAreMissingAfterSuccessfulProcess(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-provider-marker")
	runner := HeadlessRunner{Command: writeQwenScript(t, "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' 'API Error: 403 permission_error usage limit'\n")}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatal("expected runner error")
	}

	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runtime runner error, got %T: %v", err, err)
	}
	if got, want := runnerErr.Code, acpruntime.ErrorCodeRunnerUnavailable; got != want {
		t.Fatalf("expected %s, got %s (%v)", want, got, err)
	}
	if runnerErr.Stdout == "" || !containsAll(runnerErr.Stdout, []string{"403", "permission_error"}) {
		t.Fatalf("expected provider-unavailable markers in stdout, got %q", runnerErr.Stdout)
	}
}

func TestRunClassifiesSilentMissingArtifactsAsProviderUnavailable(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-silent-missing")
	runner := HeadlessRunner{Command: writeQwenScript(t, "#!/usr/bin/env bash\nset -eu\n")}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatal("expected runner error")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runtime runner error, got %T: %v", err, err)
	}
	if got, want := runnerErr.Code, acpruntime.ErrorCodeRunnerUnavailable; got != want {
		t.Fatalf("expected %s, got %s (%v)", want, got, err)
	}
}

func TestRunKeepsMalformedManifestAsRuntimeContractFailure(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-malformed")
	script := "#!/usr/bin/env bash\nset -eu\nmkdir -p " + shellQuote(task.WriteRoot) + "\nprintf '%s\\n' '{\"version\":1,\"run_id\":\"" + task.RunID + "\",\"step_id\":\"init.step0.constitution\",\"step_contract\":\"constitution\",\"agent_role\":\"architect\",\"manifest_version\":2,\"outputs\":[]}' > " + shellQuote(filepath.Join(task.WriteRoot, "constitution-draft.json")) + "\nprintf '%s\\n' 'API Error: 403 permission_error usage limit'\n"
	runner := HeadlessRunner{Command: writeQwenScript(t, script)}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatal("expected runtime contract error")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runtime runner error, got %T: %v", err, err)
	}
	if got, want := runnerErr.Code, acpruntime.ErrorCodeRuntimeContract; got != want {
		t.Fatalf("expected %s, got %s (%v)", want, got, err)
	}
}

func containsAll(text string, tokens []string) bool {
	for _, token := range tokens {
		if !strings.Contains(text, token) {
			return false
		}
	}
	return true
}

func newQwenDraftTask(t *testing.T, runID string) acpruntime.Task {
	t.Helper()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", runID, "constitution")
	draftRoot := filepath.Join(workspace, "reports", "taskruns", runID, "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	return acpruntime.Task{
		TaskID:            "task-" + runID,
		RunID:             runID,
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
		StartedAtUTC:      time.Now().UTC(),
	}
}

func writeQwenScript(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "qwen-stub.sh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write qwen stub: %v", err)
	}
	return path
}

func qwenValidDraftScript(task acpruntime.Task, tail string) string {
	return `#!/usr/bin/env bash
set -eu
write_root=` + shellQuote(task.WriteRoot) + `
draft_root=` + shellQuote(task.DraftFinalRoot) + `
run_id=` + shellQuote(task.RunID) + `
mkdir -p "$write_root" "$draft_root"
cat >"$write_root/constitution-draft.json" <<EOF
{
  "version": 1,
  "run_id": "$run_id",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "summary": "Recovered constitution artifacts.",
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
` + tail + "\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
