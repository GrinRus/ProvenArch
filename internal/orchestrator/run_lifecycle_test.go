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

func TestRunWithIDTerminalGuardPersistsFailedHistoryOnPanic(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(panicRunner{}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time {
			return time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
		}),
	)

	panicked := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicked = true
			}
		}()
		_, _, _ = service.Run(context.Background(), RunRequest{
			Workspace:      ws,
			Pipeline:       PipelineInit,
			NonInteractive: true,
		})
	}()
	if !panicked {
		t.Fatal("expected runner panic to propagate")
	}

	history := readRunHistorySnapshot(t, ws.Path)
	if len(history.Items) != 1 {
		t.Fatalf("expected one history item, got %d", len(history.Items))
	}
	item := history.Items[0]
	if item.Status != RunStatusFailed {
		t.Fatalf("expected failed history status, got %s", item.Status)
	}
	if item.FinishedAt == nil || strings.TrimSpace(*item.FinishedAt) == "" {
		t.Fatalf("expected terminal finished_at in history: %#v", item)
	}
	if item.ErrorCode != "internal_failure" {
		t.Fatalf("expected internal_failure error_code, got %q", item.ErrorCode)
	}
	if !strings.Contains(item.Error, "run panic") {
		t.Fatalf("expected panic message in history error, got %q", item.Error)
	}
}

func TestTerminalGuardPersistsFailedHistoryOnContextCancellation(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time {
			return time.Date(2026, 5, 7, 11, 0, 0, 0, time.UTC)
		}),
	)
	runID := "run_cancel_guard"
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:     runID,
			Pipeline:  string(PipelineRefresh),
			Status:    RunStatusRunning,
			StartedAt: time.Date(2026, 5, 7, 10, 59, 0, 0, time.UTC),
		},
	})

	service.terminalizeActiveRunAfterUnexpectedExit(runID, context.Canceled, "run failed: context canceled")

	history := readRunHistorySnapshot(t, ws.Path)
	item, ok := findHistoryItem(history, runID)
	if !ok {
		t.Fatalf("expected history item %q", runID)
	}
	if item.Status != RunStatusFailed {
		t.Fatalf("expected failed history status, got %s", item.Status)
	}
	if item.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected %s error_code, got %q", runErrorCodeCanceled, item.ErrorCode)
	}
	if item.FinishedAt == nil {
		t.Fatal("expected finished_at")
	}
}

func TestClassifyExecutionErrorMapsPlainContextFailures(t *testing.T) {
	t.Parallel()

	code, _ := classifyExecutionError(context.Canceled)
	if code != runErrorCodeCanceled {
		t.Fatalf("expected %s for context.Canceled, got %q", runErrorCodeCanceled, code)
	}

	code, _ = classifyExecutionError(context.DeadlineExceeded)
	if code != string(acpruntime.ErrorCodeRuntimeTimeout) {
		t.Fatalf("expected %s for context.DeadlineExceeded, got %q", acpruntime.ErrorCodeRuntimeTimeout, code)
	}
}

type panicRunner struct{}

func (panicRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	panic("boom")
}

func readRunHistorySnapshot(t *testing.T, root string) runHistorySnapshot {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, runHistoryPath))
	if err != nil {
		t.Fatalf("read run history: %v", err)
	}
	var snapshot runHistorySnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("parse run history: %v", err)
	}
	return snapshot
}

func findHistoryItem(snapshot runHistorySnapshot, runID string) (runHistoryItem, bool) {
	for _, item := range snapshot.Items {
		if item.RunID == runID {
			return item, true
		}
	}
	return runHistoryItem{}, false
}
