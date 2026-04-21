package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/qwencode"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRecordedRuntimeToReportChainSalvageAndClassification(t *testing.T) {
	t.Parallel()

	ws := createSingleRepoWorkspace(t)
	commandPath := writeRecordedQwenChainCommand(t)
	service := NewService(
		WithRunner(qwencode.HeadlessRunner{Command: commandPath}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC) }),
	)

	initInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}, "init-run")
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if initInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected init run to succeed, got %s (%s)", initInfo.Status, initInfo.Error)
	}

	batchRunDir := filepath.Join(t.TempDir(), "run1")
	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "init-run", "reports")); err != nil {
		t.Fatalf("snapshot init reports: %v", err)
	}

	refreshInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	}, "refresh-run")
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if refreshInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected refresh run to succeed, got %s (%s)", refreshInfo.Status, refreshInfo.Error)
	}

	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "refresh-run", "reports")); err != nil {
		t.Fatalf("snapshot refresh reports: %v", err)
	}
	if err := copyTree(ws.Path, filepath.Join(batchRunDir, "arch-workspace")); err != nil {
		t.Fatalf("copy arch workspace: %v", err)
	}

	if err := writeRecordedIntegrationBatchFiles(batchRunDir, ws.Path, "qwen-code"); err != nil {
		t.Fatalf("write synthetic batch files: %v", err)
	}

	result, err := evaluateRunWithBatchReport(t, batchRunDir, "qwen-code")
	if err != nil {
		t.Fatalf("evaluate run with batch report: %v", err)
	}
	if result.FailureClass != "none" {
		t.Fatalf("expected no failure class, got %+v", result)
	}
	if !result.HardPass {
		t.Fatalf("expected hard_pass=true, got %+v", result)
	}
	if result.ArtifactSource != "snapshot" {
		t.Fatalf("expected snapshot artifact source, got %+v", result)
	}
}

func TestRecordedRuntimeToReportChainArtifactRepairAndClassification(t *testing.T) {
	t.Parallel()

	ws := createSingleRepoWorkspace(t)
	commandPath := writeRecordedQwenArtifactRepairCommand(t)
	service := NewService(
		WithRunner(qwencode.HeadlessRunner{Command: commandPath}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC) }),
	)

	initInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}, "init-run")
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if initInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected init run to succeed, got %s (%s)", initInfo.Status, initInfo.Error)
	}

	refreshInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	}, "refresh-run")
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if refreshInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected refresh run to succeed, got %s (%s)", refreshInfo.Status, refreshInfo.Error)
	}

	metaFiles, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "raw", "*-prompt-meta.json"))
	if err != nil {
		t.Fatalf("glob prompt metadata: %v", err)
	}
	foundArtifactRepair := false
	for _, metaPath := range metaFiles {
		rawMeta, err := os.ReadFile(metaPath)
		if err != nil {
			t.Fatalf("read prompt metadata %q: %v", metaPath, err)
		}
		meta := map[string]any{}
		if err := json.Unmarshal(rawMeta, &meta); err != nil {
			t.Fatalf("parse prompt metadata %q: %v", metaPath, err)
		}
		if strings.TrimSpace(fmt.Sprintf("%v", meta["attempt"])) == "artifact-repair" {
			foundArtifactRepair = true
			break
		}
	}
	if !foundArtifactRepair {
		t.Fatalf("expected recorded integration to materialize artifact-repair prompt artifacts")
	}

	batchRunDir := filepath.Join(t.TempDir(), "run1")
	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "init-run", "reports")); err != nil {
		t.Fatalf("snapshot init reports: %v", err)
	}
	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "refresh-run", "reports")); err != nil {
		t.Fatalf("snapshot refresh reports: %v", err)
	}
	if err := copyTree(ws.Path, filepath.Join(batchRunDir, "arch-workspace")); err != nil {
		t.Fatalf("copy arch workspace: %v", err)
	}
	if err := writeRecordedIntegrationBatchFiles(batchRunDir, ws.Path, "qwen-code"); err != nil {
		t.Fatalf("write synthetic batch files: %v", err)
	}

	result, err := evaluateRunWithBatchReport(t, batchRunDir, "qwen-code")
	if err != nil {
		t.Fatalf("evaluate run with batch report: %v", err)
	}
	if result.FailureClass != "none" || !result.HardPass {
		t.Fatalf("expected artifact-repair recorded chain to hard-pass, got %+v", result)
	}
}

func TestRecordedRuntimeToReportChainRunnerUnavailableClassification(t *testing.T) {
	t.Parallel()

	ws := createSingleRepoWorkspace(t)
	commandPath := writeRecordedQwenUnavailableCommand(t)
	service := NewService(
		WithRunner(qwencode.HeadlessRunner{Command: commandPath}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC) }),
	)

	initInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}, "init-run")
	if err == nil {
		t.Fatalf("expected init pipeline to fail for runner_unavailable path")
	}
	if initInfo.Status != RunStatusFailed {
		t.Fatalf("expected init run to fail, got %s (%s)", initInfo.Status, initInfo.Error)
	}

	batchRunDir := filepath.Join(t.TempDir(), "run1")
	reportsRoot := filepath.Join(ws.Path, "reports")
	if _, err := os.Stat(reportsRoot); err == nil {
		if err := copyTree(reportsRoot, filepath.Join(batchRunDir, "snapshots", "init-run", "reports")); err != nil {
			t.Fatalf("snapshot init reports: %v", err)
		}
	}
	if err := copyTree(ws.Path, filepath.Join(batchRunDir, "arch-workspace")); err != nil {
		t.Fatalf("copy arch workspace: %v", err)
	}
	if err := writeRecordedFailureBatchFiles(batchRunDir, "qwen-code"); err != nil {
		t.Fatalf("write failed synthetic batch files: %v", err)
	}

	result, err := evaluateRunWithBatchReport(t, batchRunDir, "qwen-code")
	if err != nil {
		t.Fatalf("evaluate run with batch report: %v", err)
	}
	if result.FailureClass != "runner_unavailable" {
		t.Fatalf("expected runner_unavailable classification, got %+v", result)
	}
	if result.HardPass {
		t.Fatalf("expected hard_pass=false for unavailable provider, got %+v", result)
	}
	if !containsIssue(result.Issues, "reliability:runner-unavailable-quota") {
		t.Fatalf("expected quota availability issue in report, got %+v", result)
	}
}

func TestRecordedClaudeRuntimeToReportChainSalvageAndClassification(t *testing.T) {
	t.Parallel()

	ws := createSingleRepoWorkspace(t)
	commandPath := writeRecordedClaudeChainCommand(t)
	service := NewService(
		WithRunner(claudecode.HeadlessRunner{Command: commandPath}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC) }),
	)

	initInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}, "init-run")
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if initInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected init run to succeed, got %s (%s)", initInfo.Status, initInfo.Error)
	}

	batchRunDir := filepath.Join(t.TempDir(), "run1")
	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "init-run", "reports")); err != nil {
		t.Fatalf("snapshot init reports: %v", err)
	}

	refreshInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	}, "refresh-run")
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if refreshInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected refresh run to succeed, got %s (%s)", refreshInfo.Status, refreshInfo.Error)
	}

	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "refresh-run", "reports")); err != nil {
		t.Fatalf("snapshot refresh reports: %v", err)
	}
	if err := copyTree(ws.Path, filepath.Join(batchRunDir, "arch-workspace")); err != nil {
		t.Fatalf("copy arch workspace: %v", err)
	}

	if err := writeRecordedIntegrationBatchFiles(batchRunDir, ws.Path, "claude-code"); err != nil {
		t.Fatalf("write synthetic batch files: %v", err)
	}

	result, err := evaluateRunWithBatchReport(t, batchRunDir, "claude-code")
	if err != nil {
		t.Fatalf("evaluate run with batch report: %v", err)
	}
	if result.FailureClass != "none" {
		t.Fatalf("expected no failure class, got %+v", result)
	}
	if !result.HardPass {
		t.Fatalf("expected hard_pass=true, got %+v", result)
	}
	if result.ArtifactSource != "snapshot" {
		t.Fatalf("expected snapshot artifact source, got %+v", result)
	}
}

func TestRecordedClaudeRuntimeToReportChainArtifactRepairAndClassification(t *testing.T) {
	t.Parallel()

	ws := createSingleRepoWorkspace(t)
	commandPath := writeRecordedClaudeArtifactRepairCommand(t)
	service := NewService(
		WithRunner(claudecode.HeadlessRunner{Command: commandPath}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return time.Date(2026, 4, 3, 14, 30, 0, 0, time.UTC) }),
	)

	initInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}, "init-run")
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if initInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected init run to succeed, got %s (%s)", initInfo.Status, initInfo.Error)
	}

	refreshInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	}, "refresh-run")
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if refreshInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected refresh run to succeed, got %s (%s)", refreshInfo.Status, refreshInfo.Error)
	}

	metaFiles, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "raw", "*-prompt-meta.json"))
	if err != nil {
		t.Fatalf("glob prompt metadata: %v", err)
	}
	foundArtifactRepair := false
	for _, metaPath := range metaFiles {
		rawMeta, err := os.ReadFile(metaPath)
		if err != nil {
			t.Fatalf("read prompt metadata %q: %v", metaPath, err)
		}
		meta := map[string]any{}
		if err := json.Unmarshal(rawMeta, &meta); err != nil {
			t.Fatalf("parse prompt metadata %q: %v", metaPath, err)
		}
		if strings.TrimSpace(fmt.Sprintf("%v", meta["attempt"])) == "artifact-repair" {
			foundArtifactRepair = true
			break
		}
	}
	if !foundArtifactRepair {
		t.Fatalf("expected recorded integration to materialize artifact-repair prompt artifacts")
	}

	batchRunDir := filepath.Join(t.TempDir(), "run1")
	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "init-run", "reports")); err != nil {
		t.Fatalf("snapshot init reports: %v", err)
	}
	if err := copyTree(filepath.Join(ws.Path, "reports"), filepath.Join(batchRunDir, "snapshots", "refresh-run", "reports")); err != nil {
		t.Fatalf("snapshot refresh reports: %v", err)
	}
	if err := copyTree(ws.Path, filepath.Join(batchRunDir, "arch-workspace")); err != nil {
		t.Fatalf("copy arch workspace: %v", err)
	}
	if err := writeRecordedIntegrationBatchFiles(batchRunDir, ws.Path, "claude-code"); err != nil {
		t.Fatalf("write synthetic batch files: %v", err)
	}

	result, err := evaluateRunWithBatchReport(t, batchRunDir, "claude-code")
	if err != nil {
		t.Fatalf("evaluate run with batch report: %v", err)
	}
	if result.FailureClass != "none" || !result.HardPass {
		t.Fatalf("expected artifact-repair recorded chain to hard-pass, got %+v", result)
	}
}

func TestRecordedClaudeRuntimeToReportChainRunnerUnavailableClassification(t *testing.T) {
	t.Parallel()

	ws := createSingleRepoWorkspace(t)
	commandPath := writeRecordedClaudeUnavailableCommand(t)
	service := NewService(
		WithRunner(claudecode.HeadlessRunner{Command: commandPath}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return time.Date(2026, 4, 3, 15, 0, 0, 0, time.UTC) }),
	)

	initInfo, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}, "init-run")
	if err == nil {
		t.Fatalf("expected init pipeline to fail for runner_unavailable path")
	}
	if initInfo.Status != RunStatusFailed {
		t.Fatalf("expected init run to fail, got %s (%s)", initInfo.Status, initInfo.Error)
	}

	batchRunDir := filepath.Join(t.TempDir(), "run1")
	reportsRoot := filepath.Join(ws.Path, "reports")
	if _, err := os.Stat(reportsRoot); err == nil {
		if err := copyTree(reportsRoot, filepath.Join(batchRunDir, "snapshots", "init-run", "reports")); err != nil {
			t.Fatalf("snapshot init reports: %v", err)
		}
	}
	if err := copyTree(ws.Path, filepath.Join(batchRunDir, "arch-workspace")); err != nil {
		t.Fatalf("copy arch workspace: %v", err)
	}
	if err := writeRecordedFailureBatchFiles(batchRunDir, "claude-code"); err != nil {
		t.Fatalf("write failed synthetic batch files: %v", err)
	}

	result, err := evaluateRunWithBatchReport(t, batchRunDir, "claude-code")
	if err != nil {
		t.Fatalf("evaluate run with batch report: %v", err)
	}
	if result.FailureClass != "runner_unavailable" {
		t.Fatalf("expected runner_unavailable classification, got %+v", result)
	}
	if result.HardPass {
		t.Fatalf("expected hard_pass=false for unavailable provider, got %+v", result)
	}
	if !containsIssue(result.Issues, "reliability:runner-unavailable-quota") {
		t.Fatalf("expected quota availability issue in report, got %+v", result)
	}
}

type batchEvalResult struct {
	HardPass       bool     `json:"hard_pass"`
	FailureClass   string   `json:"failure_class"`
	ArtifactSource string   `json:"artifact_source"`
	Issues         []string `json:"issues"`
	IssueDetails   []string `json:"issue_details"`
}

func evaluateRunWithBatchReport(t *testing.T, runDir string, provider string) (batchEvalResult, error) {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return batchEvalResult{}, err
	}
	script := `
import importlib.util
import json
import pathlib
import sys

repo_root = pathlib.Path(sys.argv[1])
run_dir = pathlib.Path(sys.argv[2])
spec = importlib.util.spec_from_file_location("e2e_batch_report", repo_root / "scripts" / "e2e_batch_report.py")
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
sys.modules[spec.name] = module
spec.loader.exec_module(module)
preflight = {
    "expected_repo_count": 1,
    "declared_repos_meta": {
        "expected_repo_count": 1,
        "declared_repos": [
            {
                "source": "path",
                "path": str(run_dir / "arch-workspace" / "repos" / "payments-service"),
            }
        ],
    },
}
result = module.evaluate_run(sys.argv[3], 1, run_dir, preflight, None)
print(json.dumps({
    "hard_pass": result.hard_pass,
    "failure_class": result.failure_class,
    "artifact_source": result.artifact_source,
    "issues": result.issues,
    "issue_details": result.issue_details,
}))
`
	cmd := exec.Command("python3", "-c", script, repoRoot, runDir, provider)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return batchEvalResult{}, fmt.Errorf("python batch evaluation failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	var result batchEvalResult
	if err := json.Unmarshal(output, &result); err != nil {
		return batchEvalResult{}, fmt.Errorf("decode batch evaluation: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return result, nil
}

func writeRecordedIntegrationBatchFiles(runDir string, workspacePath string, provider string) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	summary := strings.Join([]string{
		"# Session Summary",
		"",
		"- result: passed",
		"- quality_gates: passed",
		"- failure_reason: none",
		"- expected_runs: 2",
		"- completed_runs: 2",
		"- expected_headless_runs: 2",
		"- completed_headless_runs: 2",
		"- running_runs_detected: 0",
		"- termination_signal: none",
		"",
		"## API Simulation",
		"- status: succeeded",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(runDir, "session-summary.md"), []byte(summary), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "full-run.log"), []byte("recorded integration batch completed successfully\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "batch-driver.log"), []byte("driver completed with process_exit=0\n"), 0o644); err != nil {
		return err
	}

	rows := []string{}
	for _, item := range []struct {
		RunID    string
		Pipeline string
	}{
		{RunID: "init-run", Pipeline: "init"},
		{RunID: "refresh-run", Pipeline: "refresh"},
	} {
		quality, err := readQualitySnapshot(filepath.Join(workspacePath, "reports", "taskruns", item.RunID+"-quality.json"))
		if err != nil {
			return err
		}
		rows = append(rows, strings.Join([]string{
			"1",
			"headless",
			provider,
			item.Pipeline,
			item.RunID,
			"succeeded",
			fmt.Sprintf("%d", quality.Totals.SignalScore),
			fmt.Sprintf("%d", quality.Totals.ChangesetOps),
			fmt.Sprintf("%d", quality.Totals.FindingsAdded),
			fmt.Sprintf("%d", quality.Totals.QuestionsCount),
			fmt.Sprintf("%d", quality.Totals.CoverageObserved),
			fmt.Sprintf("%d", quality.Totals.CoverageMissing),
			fmt.Sprintf("%d", quality.Totals.WarningsCount),
			strings.Join(quality.RuntimeVersions, ","),
			"reports/taskruns/" + item.RunID + "-quality.json",
			"reports",
		}, "\t"))
	}
	return os.WriteFile(filepath.Join(runDir, "run-results.tsv"), []byte(strings.Join(rows, "\n")+"\n"), 0o644)
}

func writeRecordedFailureBatchFiles(runDir string, provider string) error {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	summary := strings.Join([]string{
		"# Session Summary",
		"",
		"- result: failed",
		"- quality_gates: skipped",
		"- failure_reason: runner_unavailable",
		"- expected_runs: 1",
		"- completed_runs: 1",
		"- expected_headless_runs: 1",
		"- completed_headless_runs: 1",
		"- running_runs_detected: 0",
		"- termination_signal: none",
		"",
		"## API Simulation",
		"- status: succeeded",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(runDir, "session-summary.md"), []byte(summary), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "full-run.log"), []byte("recorded integration batch failed due to runner_unavailable\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "batch-driver.log"), []byte("driver completed with process_exit=1\n"), 0o644); err != nil {
		return err
	}
	row := strings.Join([]string{
		"1",
		"headless",
		provider,
		"init",
		"init-run",
		"failed",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		provider + "@recorded-integration",
		"reports/taskruns/init-run-quality.json",
		"reports",
	}, "\t")
	return os.WriteFile(filepath.Join(runDir, "run-results.tsv"), []byte(row+"\n"), 0o644)
}

type qualitySnapshot struct {
	RuntimeVersions []string `json:"runtime_versions"`
	Totals          struct {
		ChangesetOps     int `json:"changeset_ops"`
		FindingsAdded    int `json:"findings_added"`
		QuestionsCount   int `json:"questions_count"`
		CoverageObserved int `json:"coverage_observed"`
		CoverageMissing  int `json:"coverage_missing"`
		WarningsCount    int `json:"warnings_count"`
		SignalScore      int `json:"signal_score"`
	} `json:"totals"`
}

func readQualitySnapshot(path string) (qualitySnapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return qualitySnapshot{}, err
	}
	var snapshot qualitySnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return qualitySnapshot{}, err
	}
	return snapshot, nil
}

func writeRecordedQwenChainCommand(t *testing.T) string {
	t.Helper()
	return writeRecordedProviderChainCommand(t, "qwen", "--prompt", "qwen-code")
}

func writeRecordedQwenUnavailableCommand(t *testing.T) string {
	t.Helper()
	return writeRecordedProviderUnavailableCommand(t, "qwen", "qwen-code")
}

func writeRecordedQwenArtifactRepairCommand(t *testing.T) string {
	t.Helper()
	return writeRecordedProviderArtifactRepairCommand(t, "qwen", "--prompt", "qwen-code")
}

func writeRecordedClaudeChainCommand(t *testing.T) string {
	t.Helper()
	return writeRecordedProviderChainCommand(t, "claude", "-p", "claude-code")
}

func writeRecordedClaudeUnavailableCommand(t *testing.T) string {
	t.Helper()
	return writeRecordedProviderUnavailableCommand(t, "claude", "claude-code")
}

func writeRecordedClaudeArtifactRepairCommand(t *testing.T) string {
	t.Helper()
	return writeRecordedProviderArtifactRepairCommand(t, "claude", "-p", "claude-code")
}

func writeRecordedProviderChainCommand(t *testing.T, fileName string, promptFlag string, provider string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), fileName)
	script := fmt.Sprintf(`#!/bin/sh
set -eu
prompt=""
prev=""
for arg in "$@"; do
  if [ "$prev" = %q ]; then
    prompt="$arg"
    break
  fi
  prev="$arg"
done
if [ -z "$prompt" ]; then
  echo "missing prompt payload" >&2
  exit 1
fi
PROMPT="$prompt" python3 - <<'PY'
import json
import os
import pathlib
import re

prompt = os.environ["PROMPT"]
task_id_match = re.search(r'"task_id"\s*:\s*"([^"]+)"', prompt)
step_id_match = re.search(r'"step_id"\s*:\s*"([^"]+)"', prompt)
run_id_match = re.search(r'"run_id"\s*:\s*"([^"]+)"', prompt)
write_root_match = re.search(r'write_root \(absolute\) = "([^"]+)"', prompt)
task_id = task_id_match.group(1) if task_id_match else ""
step_id = step_id_match.group(1) if step_id_match else ""
run_id = run_id_match.group(1) if run_id_match else ""
write_root = write_root_match.group(1) if write_root_match else ""

if step_id.endswith("step1.collect"):
    root = pathlib.Path(write_root)
    root.mkdir(parents=True, exist_ok=True)
    (root / "service-catalog.md").write_text("# Service Catalog\n\n- Service: payments-service\n", encoding="utf-8")
    manifest = {
        "version": 1,
        "run_id": run_id,
        "step_id": step_id,
        "shard_id": root.name,
        "agent_role": "shard-analyst",
        "artifact_root": write_root,
        "repo_scopes": ["payments-service"],
        "path_scopes": ["."],
        "summary": "Recorded collect manifest for deterministic salvage.",
        "documents": [
            {
                "id": "doc.service-catalog",
                "kind": "report",
                "title": "Service Catalog",
                "path": "service-catalog.md",
                "canonical_path": "reports/as-is/service-catalog.md",
                "topics": ["services", "architecture"],
                "citation_ids": ["cite.payments.readme"],
                "status": "staged",
            }
        ],
        "citations": [
            {
                "id": "cite.payments.readme",
                "repo": "payments-service",
                "path": "README.md",
                "claim_ids": ["claim.payments.readme"],
                "document_ids": ["doc.service-catalog"],
            }
        ],
        "compatibility": {
            "coverage": {
                "observed": ["service catalog"],
                "missing": ["owner mappings", "runtime metrics", "dependency graph"],
                "notes": ["recorded integration salvage"],
            },
            "questions": [],
            "entities": [],
            "edges": [],
            "findings": [],
        },
    }
    (root / "shard-pack-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print("recorded event-stream chatter without a top-level taskresult")
else:
    payload = {
        "meta": {
            "task_id": task_id,
            "step_id": step_id,
            "runtime": {"name": %q, "version": "recorded-integration"},
            "started_at": "2026-04-03T12:00:00Z",
        },
        "summary": "Recorded findings analysis completed.",
        "changeset": [
            {
                "op": "add_finding",
                "finding": {
                    "id": "finding.payments.owner-gap",
                    "severity": "medium",
                    "title": "Owner mapping requires confirmation",
                    "description": "Recorded integration fixture keeps one deterministic finding.",
                    "rule_id": "ownership.coverage",
                    "related_ids": ["payments-service"],
                    "provenance": {
                        "kind": "observation",
                        "confidence": 0.82,
                        "evidence": [{"repo": "payments-service", "path": "README.md"}],
                    },
                },
            }
        ],
        "questions": [
            {
                "id": "q.refresh.delta",
                "text": "Which runtime metrics still need to be captured?",
            }
        ],
        "coverage": {
            "observed": ["service catalog"],
            "missing": ["owner mappings", "runtime metrics", "dependency graph"],
            "notes": ["recorded integration findings"],
        },
    }
    print(json.dumps([
        {
            "type": "assistant",
            "message": {
                "content": [
                    {
                        "type": "text",
                        "text": json.dumps(payload),
                    }
                ]
            }
        }
    ]))
PY
`, promptFlag, provider)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write recorded %s chain command: %v", provider, err)
	}
	return path
}

func writeRecordedProviderUnavailableCommand(t *testing.T, fileName string, provider string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), fileName)
	script := `#!/bin/sh
set -eu
python3 - <<'PY'
import json

print(json.dumps([
    {
        "type": "assistant",
        "message": {
            "content": [
                {
                    "type": "text",
                    "text": "[API Error: 403 {\"error\":{\"type\":\"permission_error\",\"message\":\"usage limit\"}}]",
                }
            ]
        }
    },
    {
        "type": "result",
        "result": "[API Error: 403 {\"error\":{\"type\":\"permission_error\",\"message\":\"quota will be refreshed next cycle\"}}]",
    }
]))
PY
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write recorded %s unavailable command: %v", provider, err)
	}
	return path
}

func writeRecordedProviderArtifactRepairCommand(t *testing.T, fileName string, promptFlag string, provider string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), fileName)
	script := fmt.Sprintf(`#!/bin/sh
set -eu
prompt=""
prev=""
for arg in "$@"; do
  if [ "$prev" = %q ]; then
    prompt="$arg"
    break
  fi
  prev="$arg"
done
if [ -z "$prompt" ]; then
  echo "missing prompt payload" >&2
  exit 1
fi
PROMPT="$prompt" python3 - <<'PY'
import json
import os
import pathlib
import re

prompt = os.environ["PROMPT"]
task_id_match = re.search(r'"task_id"\s*:\s*"([^"]+)"', prompt)
step_id_match = re.search(r'"step_id"\s*:\s*"([^"]+)"', prompt)
run_id_match = re.search(r'"run_id"\s*:\s*"([^"]+)"', prompt)
write_root_match = re.search(r'write_root \(absolute\) = "([^"]+)"', prompt)
task_id = task_id_match.group(1) if task_id_match else ""
step_id = step_id_match.group(1) if step_id_match else ""
run_id = run_id_match.group(1) if run_id_match else ""
write_root = write_root_match.group(1) if write_root_match else ""
is_artifact_repair = "RETRY OBJECTIVE: repair collect artifacts deterministically" in prompt

def taskresult(summary):
    return {
        "meta": {
            "task_id": task_id,
            "step_id": step_id,
            "runtime": {"name": %q, "version": "recorded-integration"},
            "started_at": "2026-04-03T12:30:00Z",
        },
        "summary": summary,
        "changeset": [],
        "coverage": {
            "observed": ["service catalog"],
            "missing": ["owner mappings", "runtime metrics", "dependency graph"],
            "notes": ["recorded integration artifact repair"],
        },
    }

if step_id.endswith("step1.collect"):
    root = pathlib.Path(write_root)
    root.mkdir(parents=True, exist_ok=True)
    (root / "service-catalog.md").write_text("# Service Catalog\n\n- Service: payments-service\n", encoding="utf-8")
    manifest = {
        "version": 1,
        "run_id": run_id,
        "step_id": step_id,
        "shard_id": root.name,
        "agent_role": "shard-analyst",
        "artifact_root": write_root,
        "repo_scopes": ["payments-service"],
        "path_scopes": ["."],
        "summary": "Recorded collect manifest for artifact repair.",
        "documents": [
            {
                "id": "doc.service-catalog",
                "kind": "report",
                "title": "Service Catalog",
                "path": "service-catalog.md",
                "canonical_path": "reports/as-is/service-catalog.md",
                "topics": ["services", "architecture"],
                "citation_ids": ["cite.runtime-summary"],
                "status": "staged",
            }
        ],
        "citations": [
            {
                "id": "cite.runtime-summary",
                "repo": "",
                "path": "reports/taskruns/logs/runtime.ndjson",
                "claim_ids": ["claim.runtime-summary"],
                "document_ids": ["doc.service-catalog"],
            }
        ],
        "compatibility": {
            "coverage": {
                "observed": ["service catalog"],
                "missing": ["owner mappings", "runtime metrics", "dependency graph"],
                "notes": ["recorded integration artifact repair"],
            },
            "questions": [],
            "entities": [],
            "edges": [],
            "findings": [],
        },
    }
    if is_artifact_repair:
        manifest["documents"][0]["citation_ids"] = ["cite.payments.readme"]
        manifest["citations"][0] = {
            "id": "cite.payments.readme",
            "repo": "payments-service",
            "path": "README.md",
            "claim_ids": ["claim.payments.readme"],
            "document_ids": ["doc.service-catalog"],
        }
    (root / "shard-pack-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(taskresult("collect completed")))
else:
    print(json.dumps(taskresult("findings completed")))
PY
`, promptFlag, provider)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write recorded %s artifact repair command: %v", provider, err)
	}
	return path
}

func createSingleRepoWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repo := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# payments-service\n"), 0o644); err != nil {
		t.Fatalf("write repo readme: %v", err)
	}
	manifest := "version: 1\nrepos:\n  - name: payments-service\n    path: " + repo + "\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func copyTree(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		targetFile, err := os.Create(target)
		if err != nil {
			_ = sourceFile.Close()
			return err
		}
		if _, err := io.Copy(targetFile, sourceFile); err != nil {
			_ = sourceFile.Close()
			_ = targetFile.Close()
			return err
		}
		if err := sourceFile.Close(); err != nil {
			_ = targetFile.Close()
			return err
		}
		return targetFile.Close()
	})
}

func containsIssue(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
