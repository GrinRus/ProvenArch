package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRunWritesQualitySummaryAndQueryableLogs(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(WithHistoryWorkspace(ws))

	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded run status, got %s", info.Status)
	}

	qualityPath := "reports/taskruns/" + info.RunID + "-quality.json"
	foundQualityArtifact := false
	for _, artifact := range artifacts {
		if artifact.Path == qualityPath {
			foundQualityArtifact = true
			break
		}
	}
	if !foundQualityArtifact {
		t.Fatalf("expected quality summary artifact %q in run artifacts", qualityPath)
	}

	qualityBytes, err := os.ReadFile(filepath.Join(ws.Path, qualityPath))
	if err != nil {
		t.Fatalf("read quality summary %q: %v", qualityPath, err)
	}
	var quality struct {
		Status string `json:"status"`
		Totals struct {
			Steps       int `json:"steps"`
			SignalScore int `json:"signal_score"`
		} `json:"totals"`
		RuntimeVersions []string `json:"runtime_versions"`
	}
	if err := json.Unmarshal(qualityBytes, &quality); err != nil {
		t.Fatalf("decode quality summary: %v", err)
	}
	if quality.Status != string(RunStatusSucceeded) {
		t.Fatalf("expected quality summary status %q, got %q", RunStatusSucceeded, quality.Status)
	}
	if quality.Totals.Steps <= 0 {
		t.Fatalf("expected positive quality step count, got %d", quality.Totals.Steps)
	}
	if len(quality.RuntimeVersions) == 0 {
		t.Fatalf("expected runtime versions in quality summary")
	}

	page, ok, err := service.GetRunLogs(info.RunID, 0, 500)
	if err != nil {
		t.Fatalf("get run logs: %v", err)
	}
	if !ok {
		t.Fatalf("expected run logs for run %q", info.RunID)
	}
	if len(page.Items) == 0 {
		t.Fatalf("expected non-empty run logs page")
	}
	if page.NextCursor <= page.Items[len(page.Items)-1].Cursor {
		t.Fatalf("expected next_cursor > last cursor, got next=%d last=%d", page.NextCursor, page.Items[len(page.Items)-1].Cursor)
	}
	foundRuntimeCompletion := false
	for _, entry := range page.Items {
		if strings.Contains(entry.Message, "runtime task completed") {
			foundRuntimeCompletion = true
			break
		}
	}
	if !foundRuntimeCompletion {
		t.Fatalf("expected runtime completion event in first logs page")
	}

	nextPage, ok, err := service.GetRunLogs(info.RunID, page.NextCursor, 10)
	if err != nil {
		t.Fatalf("get next run logs page: %v", err)
	}
	if !ok {
		t.Fatalf("expected run logs for run %q on next page", info.RunID)
	}
	for _, entry := range nextPage.Items {
		if entry.Cursor < page.NextCursor {
			t.Fatalf("expected cursor >= %d in next page, got %d", page.NextCursor, entry.Cursor)
		}
	}
}

func TestRunAggregatesTaskResultWarningsIntoRunWarningsAndLogs(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(syntheticWarningRunner{}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline with warning runner: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded run status, got %s", info.Status)
	}
	if len(info.Warnings) == 0 {
		t.Fatalf("expected run warnings to include runtime warnings")
	}
	foundRuntimeWarning := false
	for _, warning := range info.Warnings {
		if strings.Contains(warning, "synthetic runtime warning") {
			foundRuntimeWarning = true
			break
		}
	}
	if !foundRuntimeWarning {
		t.Fatalf("expected synthetic runtime warning in run warnings, got %+v", info.Warnings)
	}

	page, ok, err := service.GetRunLogs(info.RunID, 0, 500)
	if err != nil {
		t.Fatalf("get run logs: %v", err)
	}
	if !ok {
		t.Fatalf("expected run logs for run %q", info.RunID)
	}
	hasWarningLevel := false
	for _, item := range page.Items {
		if item.Level == RunLogLevelWarning {
			hasWarningLevel = true
			break
		}
	}
	if !hasWarningLevel {
		t.Fatalf("expected warning-level log entries in run logs")
	}
}

func TestRunQualitySummaryRuntimeVersionsOmitTrailingAtWhenVersionIsEmpty(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(syntheticNoVersionRunner{}),
	)

	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline with no-version runner: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded run status, got %s", info.Status)
	}

	qualityPath := "reports/taskruns/" + info.RunID + "-quality.json"
	foundQualityArtifact := false
	for _, artifact := range artifacts {
		if artifact.Path == qualityPath {
			foundQualityArtifact = true
			break
		}
	}
	if !foundQualityArtifact {
		t.Fatalf("expected quality summary artifact %q in run artifacts", qualityPath)
	}

	qualityBytes, err := os.ReadFile(filepath.Join(ws.Path, qualityPath))
	if err != nil {
		t.Fatalf("read quality summary %q: %v", qualityPath, err)
	}
	var quality struct {
		RuntimeVersions []string `json:"runtime_versions"`
	}
	if err := json.Unmarshal(qualityBytes, &quality); err != nil {
		t.Fatalf("decode quality summary: %v", err)
	}
	if len(quality.RuntimeVersions) == 0 {
		t.Fatalf("expected runtime versions in quality summary")
	}
	for _, version := range quality.RuntimeVersions {
		if strings.HasSuffix(version, "@") {
			t.Fatalf("runtime version must not have trailing '@': %q", version)
		}
	}
	if quality.RuntimeVersions[0] != "synthetic-headless" {
		t.Fatalf("expected runtime_versions to contain bare runtime name, got %+v", quality.RuntimeVersions)
	}
}

func TestRunQualitySummaryRuntimeVersionsPreferVersionedEntry(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(syntheticMixedVersionRunner{}),
	)

	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline with mixed-version runner: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded run status, got %s", info.Status)
	}

	qualityPath := "reports/taskruns/" + info.RunID + "-quality.json"
	foundQualityArtifact := false
	for _, artifact := range artifacts {
		if artifact.Path == qualityPath {
			foundQualityArtifact = true
			break
		}
	}
	if !foundQualityArtifact {
		t.Fatalf("expected quality summary artifact %q in run artifacts", qualityPath)
	}

	qualityBytes, err := os.ReadFile(filepath.Join(ws.Path, qualityPath))
	if err != nil {
		t.Fatalf("read quality summary %q: %v", qualityPath, err)
	}
	var quality struct {
		RuntimeVersions []string `json:"runtime_versions"`
	}
	if err := json.Unmarshal(qualityBytes, &quality); err != nil {
		t.Fatalf("decode quality summary: %v", err)
	}
	if len(quality.RuntimeVersions) != 1 {
		t.Fatalf("expected single runtime version entry, got %+v", quality.RuntimeVersions)
	}
	if quality.RuntimeVersions[0] != "synthetic-headless@v1" {
		t.Fatalf("expected versioned runtime entry to win over bare name, got %+v", quality.RuntimeVersions)
	}
}

func TestCleanupRunLogsByTTLAndMaxRuns(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return now }),
		WithRunLogsRetention(2*time.Hour, 2),
	)

	service.appendRunLog("run-old", RunLogEntry{Message: "old"})
	service.appendRunLog("run-mid", RunLogEntry{Message: "mid"})
	service.appendRunLog("run-new", RunLogEntry{Message: "new"})

	oldPath, err := service.resolveRunLogPath("run-old")
	if err != nil {
		t.Fatalf("resolve old run log path: %v", err)
	}
	midPath, err := service.resolveRunLogPath("run-mid")
	if err != nil {
		t.Fatalf("resolve mid run log path: %v", err)
	}
	newPath, err := service.resolveRunLogPath("run-new")
	if err != nil {
		t.Fatalf("resolve new run log path: %v", err)
	}

	if err := os.Chtimes(oldPath, now.Add(-8*time.Hour), now.Add(-8*time.Hour)); err != nil {
		t.Fatalf("set old run log mtime: %v", err)
	}
	if err := os.Chtimes(midPath, now.Add(-30*time.Minute), now.Add(-30*time.Minute)); err != nil {
		t.Fatalf("set mid run log mtime: %v", err)
	}
	if err := os.Chtimes(newPath, now.Add(-10*time.Minute), now.Add(-10*time.Minute)); err != nil {
		t.Fatalf("set new run log mtime: %v", err)
	}

	if err := service.cleanupRunLogs(); err != nil {
		t.Fatalf("cleanup run logs: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old run log to be removed by TTL, stat err=%v", err)
	}
	if _, err := os.Stat(midPath); err != nil {
		t.Fatalf("expected mid run log to remain, stat err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new run log to remain, stat err=%v", err)
	}

	service.appendRunLog("run-extra", RunLogEntry{Message: "extra"})
	extraPath, err := service.resolveRunLogPath("run-extra")
	if err != nil {
		t.Fatalf("resolve extra run log path: %v", err)
	}
	if err := os.Chtimes(extraPath, now.Add(-5*time.Minute), now.Add(-5*time.Minute)); err != nil {
		t.Fatalf("set extra run log mtime: %v", err)
	}

	if err := service.cleanupRunLogs(); err != nil {
		t.Fatalf("cleanup run logs for max-runs: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(ws.Path, runLogsPath))
	if err != nil {
		t.Fatalf("read run logs dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 run log files after max-runs cleanup, got %d", len(entries))
	}
}

type syntheticWarningRunner struct{}

func (syntheticWarningRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	payload := map[string]any{
		"meta": map[string]any{
			"task_id":    task.TaskID,
			"step_id":    task.StepID,
			"run_id":     task.RunID,
			"runtime":    map[string]any{"name": "synthetic-headless", "version": "v1"},
			"started_at": task.StartedAtUTC.Format(time.RFC3339),
		},
		"summary":   "synthetic success",
		"changeset": []any{},
		"warnings":  []string{"synthetic runtime warning"},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{RawJSON: raw}, nil
}

func (syntheticWarningRunner) Preflight(context.Context) error { return nil }

type syntheticNoVersionRunner struct{}

func (syntheticNoVersionRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	payload := map[string]any{
		"meta": map[string]any{
			"task_id":    task.TaskID,
			"step_id":    task.StepID,
			"run_id":     task.RunID,
			"runtime":    map[string]any{"name": "synthetic-headless"},
			"started_at": task.StartedAtUTC.Format(time.RFC3339),
		},
		"summary":   "synthetic success",
		"changeset": []any{},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{RawJSON: raw}, nil
}

func (syntheticNoVersionRunner) Preflight(context.Context) error { return nil }

type syntheticMixedVersionRunner struct{}

func (syntheticMixedVersionRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	runtime := map[string]any{"name": "synthetic-headless"}
	if task.StepID == "init.step4.service_findings" {
		runtime["version"] = "v1"
	}
	payload := map[string]any{
		"meta": map[string]any{
			"task_id":    task.TaskID,
			"step_id":    task.StepID,
			"run_id":     task.RunID,
			"runtime":    runtime,
			"started_at": task.StartedAtUTC.Format(time.RFC3339),
		},
		"summary":   "synthetic success",
		"changeset": []any{},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{RawJSON: raw}, nil
}

func (syntheticMixedVersionRunner) Preflight(context.Context) error { return nil }
