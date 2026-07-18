package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/refreshplan"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const asyncRunnerStartTimeout = 10 * time.Second

func TestRunPersistsRevisionImpactAndNoOpExecutionArtifacts(t *testing.T) {
	t.Parallel()
	ws := createWorkspace(t)
	for _, repo := range []string{"payments-service", "users-service"} {
		root := filepath.Join(ws.Path, "repos", repo)
		for _, args := range [][]string{{"init"}, {"add", "README.md"}, {"-c", "user.name=ACP Test", "-c", "user.email=acp@example.test", "commit", "-m", "baseline"}} {
			if output, gitErr := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); gitErr != nil {
				t.Fatalf("init git repo %s: %v: %s", repo, gitErr, output)
			}
		}
	}
	service := NewService(WithHistoryWorkspace(ws), WithClock(func() time.Time { return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC) }))
	initInfo, initArtifacts, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineInit, NonInteractive: true})
	if err != nil {
		t.Fatalf("run init: %v", err)
	}
	if initInfo.Status != RunStatusSucceeded || !artifactKindPresent(initArtifacts, "source-revisions") {
		t.Fatalf("expected source revisions in successful init inventory: info=%+v artifacts=%+v", initInfo, initArtifacts)
	}
	if _, _, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineRefresh, NonInteractive: true}); err != nil {
		t.Fatalf("establish post-init refresh baseline: %v", err)
	}
	refreshInfo, refreshArtifacts, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineRefresh, NonInteractive: true})
	if err != nil {
		t.Fatalf("run refresh: %v", err)
	}
	if refreshInfo.Status != RunStatusSucceeded || refreshInfo.CurrentStep != "" || refreshInfo.RefreshSummary == nil || refreshInfo.RefreshSummary.Mode != "no_op" {
		t.Fatalf("unchanged refresh must finish as an explained no-op: %+v", refreshInfo)
	}
	if !artifactKindPresent(refreshArtifacts, "source-revisions") || !artifactKindPresent(refreshArtifacts, "refresh-impact-plan") || !artifactKindPresent(refreshArtifacts, "refresh-execution") || !artifactKindPresent(refreshArtifacts, "refresh-materialization") {
		t.Fatalf("expected planning artifacts in refresh inventory: %+v", refreshArtifacts)
	}
	for rel, parser := range map[string]func([]byte) error{
		filepath.ToSlash(filepath.Join("reports/taskruns", refreshInfo.RunID, "source-revisions.json")):        func(raw []byte) error { _, err := refreshplan.ParseSourceRevisions(raw); return err },
		filepath.ToSlash(filepath.Join("reports/taskruns", refreshInfo.RunID, "refresh-impact-plan.json")):     func(raw []byte) error { _, err := refreshplan.ParseImpactPlan(raw); return err },
		filepath.ToSlash(filepath.Join("reports/taskruns", refreshInfo.RunID, "refresh-execution.json")):       func(raw []byte) error { _, err := refreshplan.ParseRefreshExecution(raw); return err },
		filepath.ToSlash(filepath.Join("reports/taskruns", refreshInfo.RunID, "refresh-materialization.json")): func(raw []byte) error { _, err := refreshplan.ParseRefreshMaterialization(raw); return err },
	} {
		raw, err := ws.ReadFile(rel)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if err := parser(raw); err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}
	}
}

func TestRefreshSelectivelyReplaysUnaffectedBaselineShards(t *testing.T) {
	ws := createWorkspace(t)
	for _, repo := range []string{"payments-service", "users-service"} {
		root := filepath.Join(ws.Path, "repos", repo)
		for _, args := range [][]string{{"init"}, {"add", "README.md"}, {"-c", "user.name=ACP Test", "-c", "user.email=acp@example.test", "commit", "-m", "baseline"}} {
			if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
				t.Fatalf("init git repo: %v: %s", err, output)
			}
		}
	}
	service := NewService(WithHistoryWorkspace(ws), WithClock(func() time.Time { return time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC) }))
	if _, _, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineInit, NonInteractive: true}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineRefresh, NonInteractive: true}); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(ws.Path, "repos", "payments-service")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# payments-service\n\nChanged API.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-C", repo, "add", "README.md").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", repo, "-c", "user.name=ACP Test", "-c", "user.email=acp@example.test", "commit", "-m", "incorrect intent label").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, output)
	}
	info, artifacts, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineRefresh, NonInteractive: true})
	if err != nil {
		t.Fatal(err)
	}
	if info.RefreshSummary == nil || info.RefreshSummary.Mode != "affected_only" {
		t.Fatalf("expected affected-only execution, got %+v", info.RefreshSummary)
	}
	if !artifactKindPresent(artifacts, "refresh-materialization") {
		t.Fatalf("missing materialization audit: %+v", artifacts)
	}
	raw, readErr := ws.ReadFile(filepath.ToSlash(filepath.Join("reports", "taskruns", info.RunID, "refresh-materialization.json")))
	if readErr != nil {
		t.Fatal(readErr)
	}
	materialization, parseErr := refreshplan.ParseRefreshMaterialization(raw)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	hasPreserved := false
	for _, decision := range materialization.Decisions {
		if decision.Action == "preserved" {
			hasPreserved = true
			break
		}
	}
	if !hasPreserved {
		t.Fatalf("expected at least one byte-preserved decision: %+v", materialization.Decisions)
	}
}

func artifactKindPresent(artifacts []Artifact, kind string) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			return true
		}
	}
	return false
}

func TestStartAsyncRunRequiresExplicitQueueAndSupersedesPendingRefresh(t *testing.T) {
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
	_, err = service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if !errors.Is(err, ErrRunActive) {
		t.Fatalf("expected ordinary start to return ErrRunActive, got %v", err)
	}
	secondRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
		Intent:         RunIntentQueue,
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
	thirdRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
		Intent:         RunIntentQueue,
	})
	if err != nil {
		t.Fatalf("replace pending refresh: %v", err)
	}
	superseded, ok := service.GetRun(secondRunID)
	if !ok || superseded.Status != RunStatusCanceled || superseded.ErrorCode != runErrorCodeSuperseded || superseded.SupersededByRunID != thirdRunID {
		t.Fatalf("expected typed supersession, got %+v", superseded)
	}

	close(releaseRunner)
	waitForRunTerminalInfo(t, service, firstRunID, 2*time.Second)
	waitForRunTerminalInfo(t, service, thirdRunID, 2*time.Second)
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
	waitForRunnerCalls(t, runner, 1, asyncRunnerStartTimeout)
	secondRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
		Intent:         RunIntentQueue,
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

func TestServiceShutdownCancelsActiveRunAndRejectsNewStarts(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(blockingRunner{release: make(chan struct{})}),
		WithHistoryWorkspace(ws),
	)
	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start async run: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown service: %v", err)
	}
	info := waitForRunTerminalInfo(t, service, runID, 2*time.Second)
	if info.Status != RunStatusFailed {
		t.Fatalf("expected shutdown-canceled run to fail, got %s", info.Status)
	}
	if info.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected %s, got %q", runErrorCodeCanceled, info.ErrorCode)
	}

	_, err = service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if !errors.Is(err, ErrServiceClosed) {
		t.Fatalf("expected ErrServiceClosed after shutdown, got %v", err)
	}
}

func TestServiceShutdownFailsPendingRunWithoutStartingIt(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	runner := &countingBlockingRunner{release: make(chan struct{})}
	service := NewService(
		WithRunner(runner),
		WithHistoryWorkspace(ws),
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
	waitForRunnerCalls(t, runner, 1, asyncRunnerStartTimeout)
	secondRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
		Intent:         RunIntentQueue,
	})
	if err != nil {
		t.Fatalf("queue pending run: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown service: %v", err)
	}
	waitForRunTerminalInfo(t, service, firstRunID, 2*time.Second)
	secondInfo := waitForRunTerminalInfo(t, service, secondRunID, 2*time.Second)
	if secondInfo.Status != RunStatusFailed || secondInfo.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected pending run failed/%s, got status=%s code=%q", runErrorCodeCanceled, secondInfo.Status, secondInfo.ErrorCode)
	}
	if calls := runner.callCount(); calls != 1 {
		t.Fatalf("expected only active run to start before shutdown, calls=%d", calls)
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

type countingBlockingRunner struct {
	release <-chan struct{}
	mu      sync.Mutex
	calls   int
}

func (runner *countingBlockingRunner) Run(ctx context.Context, _ acpruntime.Task) (acpruntime.Result, error) {
	runner.mu.Lock()
	runner.calls++
	runner.mu.Unlock()
	select {
	case <-ctx.Done():
		return acpruntime.Result{}, ctx.Err()
	case <-runner.release:
		return acpruntime.Result{}, nil
	}
}

func (runner *countingBlockingRunner) callCount() int {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	return runner.calls
}

func waitForRunnerCalls(t *testing.T, runner interface{ callCount() int }, minCalls int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if runner.callCount() >= minCalls {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runner did not reach %d calls before timeout; calls=%d", minCalls, runner.callCount())
}

func waitForRunTerminalInfo(t *testing.T, service *Service, runID string, timeout time.Duration) RunInfo {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		info, ok := service.GetRun(runID)
		if !ok {
			t.Fatalf("run %q not found", runID)
		}
		if info.Status == RunStatusSucceeded || info.Status == RunStatusFailed || info.Status == RunStatusCanceled {
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
