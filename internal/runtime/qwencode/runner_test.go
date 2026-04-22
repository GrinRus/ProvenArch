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

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "constitution")
	draftRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	scriptPath := filepath.Join(workspace, "fake-qwen.sh")
	script := "#!/bin/sh\nprintf '%s\\n' 'API Error: 403 permission_error usage limit'\nexit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake qwen script: %v", err)
	}

	runner := HeadlessRunner{Command: scriptPath}
	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json"},
	}

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

func TestRecoverAfterStallAcceptsValidDraftArtifacts(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "constitution")
	draftRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	writeCanonicalConstitutionDraft(t, writeRoot, draftRoot, "run-1")

	runner := HeadlessRunner{Command: "/usr/bin/true"}
	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json"},
		StartedAtUTC:      time.Now().UTC(),
	}

	recovered, _, err := runner.recoverAfterStall(
		context.Background(),
		task,
		runner.Command,
		runQwenOptions{},
		acpruntime.Result{},
		collectStallError{
			Sentinel: errDraftStalledAfterArtifacts,
			Diagnostic: collectStallDiagnostic{
				StallPhase:        collectStallPhasePostArtifact,
				ManifestState:     "valid",
				AuthoredFileCount: 2,
			},
		},
	)
	if !recovered {
		t.Fatal("expected draft stall to enter recovery path")
	}
	if err != nil {
		t.Fatalf("expected artifact-only recovery to succeed, got %v", err)
	}
}

func TestRecoverAfterStallRetriesFreshProcessWhenArtifactsWereMissing(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "constitution")
	draftRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	runner := HeadlessRunner{Command: writeQwenRetryStubRunner(t, writeRoot, draftRoot, "run-1")}
	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json"},
		StartedAtUTC:      time.Now().UTC(),
	}

	recovered, result, err := runner.recoverAfterStall(
		context.Background(),
		task,
		runner.Command,
		runQwenOptions{},
		acpruntime.Result{},
		collectStallError{
			Sentinel: errCollectStalledBeforeArtifacts,
			Diagnostic: collectStallDiagnostic{
				StallPhase:        collectStallPhasePreArtifact,
				ManifestState:     "",
				AuthoredFileCount: 0,
			},
		},
	)
	if !recovered {
		t.Fatal("expected pre-artifact stall to enter recovery path")
	}
	if err != nil {
		t.Fatalf("expected fresh retry to succeed, got %v", err)
	}
	if !strings.Contains(result.Stdout, "ok") {
		t.Fatalf("expected retry result stdout to be preserved, got %q", result.Stdout)
	}
}

func TestStallRetryOptionsDisablesCollectMonitorForPreArtifactRetry(t *testing.T) {
	t.Parallel()

	options := runQwenOptions{
		EnableCollectStallMonitor: true,
		EnableDraftStallMonitor:   true,
	}

	got := stallRetryOptions(options, collectStallDiagnostic{StallPhase: collectStallPhasePreArtifact})
	if !got.EnableCollectStallMonitor {
		t.Fatal("expected collect stall monitor to remain enabled for post-artifact recovery")
	}
	if !got.DisableCollectPreArtifactStall {
		t.Fatal("expected pre-artifact collect stall detection to be disabled for retry")
	}
	if !got.EnableDraftStallMonitor {
		t.Fatal("expected draft stall monitor to remain enabled")
	}
}

func TestStallRetryOptionsKeepsCollectMonitorForPostArtifactRetry(t *testing.T) {
	t.Parallel()

	options := runQwenOptions{
		EnableCollectStallMonitor: true,
		EnableDraftStallMonitor:   true,
	}

	got := stallRetryOptions(options, collectStallDiagnostic{StallPhase: collectStallPhasePostArtifact})
	if !got.EnableCollectStallMonitor {
		t.Fatal("expected collect stall monitor to stay enabled for post-artifact handling")
	}
	if got.DisableCollectPreArtifactStall {
		t.Fatal("expected pre-artifact stall detection to stay enabled outside pre-artifact retries")
	}
	if !got.EnableDraftStallMonitor {
		t.Fatal("expected draft stall monitor to remain enabled")
	}
}

func TestMonitorCollectStallSkipsPreArtifactWhenDisabled(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "collect")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}

	task := acpruntime.Task{WriteRoot: writeRoot}
	tracker := newCommandActivityTracker(time.Now().Add(-time.Hour))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type monitorResult struct {
		err     collectStallError
		stalled bool
	}
	done := make(chan monitorResult, 1)
	go func() {
		err, stalled := monitorCollectStall(ctx, nil, task, tracker, false)
		done <- monitorResult{err: err, stalled: stalled}
	}()

	select {
	case result := <-done:
		t.Fatalf("expected disabled pre-artifact monitor to keep waiting, got stalled=%v err=%v", result.stalled, result.err)
	case <-time.After(2500 * time.Millisecond):
	}

	cancel()

	select {
	case result := <-done:
		if result.stalled {
			t.Fatalf("expected canceled monitor to exit without stall, got %v", result.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected monitor to stop after context cancellation")
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

func writeCanonicalConstitutionDraft(t *testing.T, writeRoot string, draftRoot string, runID string) {
	t.Helper()

	manifest := `{
  "version": 1,
  "run_id": "` + runID + `",
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
}`
	if err := os.WriteFile(filepath.Join(writeRoot, "constitution-draft.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write constitution manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "charter-overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "baseline-subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write baseline subagents: %v", err)
	}
}

func writeQwenRetryStubRunner(t *testing.T, writeRoot string, draftRoot string, runID string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "qwen-retry-stub.sh")
	script := `#!/usr/bin/env bash
set -eu
write_root="` + writeRoot + `"
draft_root="` + draftRoot + `"
run_id="` + runID + `"
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
printf '%s\n' '{"type":"result","status":"ok"}'
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write retry stub: %v", err)
	}
	return path
}
