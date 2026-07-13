package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestStartAsyncRunRejectsWhenPendingOutsideDebounceWindow(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	releaseRunner := make(chan struct{})
	currentTime := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	var clockMu sync.Mutex
	clock := func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return currentTime
	}
	advanceClock := func(delta time.Duration) {
		clockMu.Lock()
		currentTime = currentTime.Add(delta)
		clockMu.Unlock()
	}

	service := NewService(
		WithRunner(blockingRunner{release: releaseRunner}),
		WithHistoryWorkspace(ws),
		WithClock(clock),
		WithDebounceWindow(time.Minute),
	)

	firstRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start first async run: %v", err)
	}

	advanceClock(10 * time.Second)
	secondRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("queue pending run inside debounce window: %v", err)
	}
	if secondRunID == firstRunID {
		t.Fatalf("expected distinct pending run id")
	}
	secondInfo, ok := service.GetRun(secondRunID)
	if !ok {
		t.Fatalf("expected pending run %q to be stored", secondRunID)
	}
	if secondInfo.Status != RunStatusQueued {
		t.Fatalf("expected second run queued, got %s", secondInfo.Status)
	}

	advanceClock(2 * time.Minute)
	_, err = service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected third async run to be rejected outside debounce window")
	}
	if !strings.Contains(err.Error(), "outside debounce window") {
		t.Fatalf("expected outside-debounce error, got %v", err)
	}

	close(releaseRunner)
	waitForRunTerminalInfo(t, service, firstRunID, 2*time.Second)
	waitForRunTerminalInfo(t, service, secondRunID, 2*time.Second)
}

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

func TestAsyncRunPanicIsTerminalAndDoesNotEscapeGoroutine(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(panicRunner{}),
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time {
			return time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
		}),
	)

	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start async run: %v", err)
	}

	info := waitForRunTerminalInfo(t, service, runID, 2*time.Second)
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed async panic run, got %s", info.Status)
	}
	if info.ErrorCode != "internal_failure" {
		t.Fatalf("expected internal_failure error code, got %q", info.ErrorCode)
	}
	if !strings.Contains(info.Error, "run panic") {
		t.Fatalf("expected run panic error, got %q", info.Error)
	}

	nextRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("service should accept a new run after async panic: %v", err)
	}
	if nextRunID == runID {
		t.Fatalf("expected distinct follow-up run id")
	}
	waitForRunTerminalInfo(t, service, nextRunID, 2*time.Second)
}

func TestAsyncRunPanicReleasesSlotAndStartsPendingRun(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	releaseFirst := make(chan struct{})
	runner := &panicFirstThenReturnRunner{releaseFirst: releaseFirst}
	service := NewService(
		WithRunner(runner),
		WithHistoryWorkspace(ws),
		WithDebounceWindow(time.Minute),
		WithClock(func() time.Time {
			return time.Date(2026, 7, 13, 12, 30, 0, 0, time.UTC)
		}),
	)

	firstRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start first async run: %v", err)
	}
	secondRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("queue pending async run: %v", err)
	}
	if secondRunID == firstRunID {
		t.Fatalf("expected distinct pending run id")
	}

	close(releaseFirst)
	firstInfo := waitForRunTerminalInfo(t, service, firstRunID, 2*time.Second)
	if firstInfo.Status != RunStatusFailed || firstInfo.ErrorCode != "internal_failure" {
		t.Fatalf("expected first run failed/internal_failure, got status=%s code=%q", firstInfo.Status, firstInfo.ErrorCode)
	}
	waitForRunTerminalInfo(t, service, secondRunID, 2*time.Second)
	if calls := runner.callCount(); calls < 2 {
		t.Fatalf("expected pending run to start after panic; runner calls=%d", calls)
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

func TestLoadHistoryFallsBackToLastGoodWhenCurrentIsMalformed(t *testing.T) {
	ws := createWorkspace(t)
	startedAt := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	service := NewService(WithHistoryWorkspace(ws))
	if err := service.storeRun(runRecord{
		info: RunInfo{
			RunID:     "run_recovered_history",
			Pipeline:  string(PipelineInit),
			Status:    RunStatusSucceeded,
			StartedAt: startedAt,
		},
		artifacts: []Artifact{{Path: "reports/as-is/overview.md", Kind: "report", Label: "Overview"}},
	}); err != nil {
		t.Fatalf("store run history: %v", err)
	}

	if err := os.WriteFile(filepath.Join(ws.Path, runHistoryPath), []byte("{malformed current history\n"), 0o644); err != nil {
		t.Fatalf("corrupt current history: %v", err)
	}

	recovered := NewService(WithHistoryWorkspace(ws))
	info, ok := recovered.GetRun("run_recovered_history")
	if !ok {
		t.Fatalf("expected run to be loaded from last-good history")
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected recovered run status %s, got %s", RunStatusSucceeded, info.Status)
	}
	if len(recovered.historyRecoveryDiagnostics) == 0 {
		t.Fatalf("expected recovery diagnostic")
	}
	if !strings.Contains(recovered.historyRecoveryDiagnostics[0], runHistoryPath+".last-good") {
		t.Fatalf("expected diagnostic to name last-good path, got %v", recovered.historyRecoveryDiagnostics)
	}
}

func TestStartAsyncRunReturnsHistoryPersistenceError(t *testing.T) {
	ws := createWorkspace(t)
	blockingHistoryRoot := filepath.Join(t.TempDir(), "history-root-file")
	if err := os.WriteFile(blockingHistoryRoot, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("create blocking history root: %v", err)
	}
	service := NewService(WithHistoryWorkspace(workspace.Root{Path: blockingHistoryRoot}))

	_, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected history persistence error")
	}
	if !strings.Contains(err.Error(), "persist run history") {
		t.Fatalf("expected persist run history error, got %v", err)
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

type panicFirstThenReturnRunner struct {
	releaseFirst <-chan struct{}
	mu           sync.Mutex
	calls        int
}

func (runner *panicFirstThenReturnRunner) Run(ctx context.Context, _ acpruntime.Task) (acpruntime.Result, error) {
	runner.mu.Lock()
	runner.calls++
	call := runner.calls
	runner.mu.Unlock()

	if call == 1 {
		select {
		case <-ctx.Done():
			return acpruntime.Result{}, ctx.Err()
		case <-runner.releaseFirst:
			panic("boom")
		}
	}
	return acpruntime.Result{}, nil
}

func (runner *panicFirstThenReturnRunner) callCount() int {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	return runner.calls
}

type blockingRunner struct {
	release <-chan struct{}
}

func (runner blockingRunner) Run(ctx context.Context, _ acpruntime.Task) (acpruntime.Result, error) {
	select {
	case <-ctx.Done():
		return acpruntime.Result{}, ctx.Err()
	case <-runner.release:
		return acpruntime.Result{}, nil
	}
}

func waitForRunTerminalInfo(t *testing.T, service *Service, runID string, timeout time.Duration) RunInfo {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		info, ok := service.GetRun(runID)
		if !ok {
			t.Fatalf("run %q not found", runID)
		}
		if info.Status == RunStatusSucceeded || info.Status == RunStatusFailed {
			return info
		}
		time.Sleep(10 * time.Millisecond)
	}
	info, ok := service.GetRun(runID)
	if !ok {
		t.Fatalf("run %q not found", runID)
	}
	t.Fatalf("run %q did not reach terminal status before timeout; last status=%s", runID, info.Status)
	return RunInfo{}
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
