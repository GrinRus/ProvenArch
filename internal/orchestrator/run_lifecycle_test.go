package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestRunSnapshotsAreDeeplyIndependent(t *testing.T) {
	t.Parallel()

	finishedAt := time.Date(2026, 7, 26, 9, 30, 0, 0, time.UTC)
	decision := acpruntime.PermissionDecision{
		RequestID: "permission-1",
		Decision:  acpruntime.PermissionDecisionNeedsUser,
		RuleID:    "managed.ui",
	}
	service := NewService()
	if err := service.storeRun(runRecord{
		info: RunInfo{
			RunID:         "run_snapshot_clone",
			Pipeline:      string(PipelineRefresh),
			Status:        RunStatusSucceeded,
			StartedAt:     finishedAt.Add(-time.Minute),
			FinishedAt:    &finishedAt,
			StepProviders: map[string]string{"refresh.step1.collect": "qwen-code"},
			Warnings:      []string{"preserved warning"},
			PendingPermissions: []acpruntime.PermissionRequest{{
				RequestID: "permission-1",
				Decision:  &decision,
			}},
			RefreshSummary: &RefreshSummary{
				Mode:        "affected_only",
				ReasonCodes: []string{"source_changed"},
			},
		},
		artifacts: []Artifact{{Path: "reports/as-is/overview.md", Kind: "report", Label: "Overview"}},
	}); err != nil {
		t.Fatalf("store run: %v", err)
	}

	assertSnapshot := func(t *testing.T) {
		t.Helper()
		info, ok := service.GetRun("run_snapshot_clone")
		if !ok {
			t.Fatal("expected stored run")
		}
		if got := info.StepProviders["refresh.step1.collect"]; got != "qwen-code" {
			t.Fatalf("step provider mutated through snapshot: %q", got)
		}
		if len(info.Warnings) != 1 || info.Warnings[0] != "preserved warning" {
			t.Fatalf("warnings mutated through snapshot: %v", info.Warnings)
		}
		if len(info.PendingPermissions) != 1 || info.PendingPermissions[0].Decision == nil || info.PendingPermissions[0].Decision.RuleID != "managed.ui" {
			t.Fatalf("permission decision mutated through snapshot: %+v", info.PendingPermissions)
		}
		if info.RefreshSummary == nil || len(info.RefreshSummary.ReasonCodes) != 1 || info.RefreshSummary.ReasonCodes[0] != "source_changed" {
			t.Fatalf("refresh summary mutated through snapshot: %+v", info.RefreshSummary)
		}
		if info.FinishedAt == nil || !info.FinishedAt.Equal(finishedAt) {
			t.Fatalf("finished_at mutated through snapshot: %v", info.FinishedAt)
		}
	}

	info, ok := service.GetRun("run_snapshot_clone")
	if !ok {
		t.Fatal("expected stored run")
	}
	info.StepProviders["refresh.step1.collect"] = "mutated"
	info.Warnings[0] = "mutated"
	info.PendingPermissions[0].Decision.RuleID = "mutated"
	info.RefreshSummary.ReasonCodes[0] = "mutated"
	*info.FinishedAt = time.Time{}
	assertSnapshot(t)

	listed := service.ListRuns(0)
	listed[0].StepProviders["refresh.step1.collect"] = "mutated again"
	listed[0].Warnings[0] = "mutated again"
	listed[0].PendingPermissions[0].Decision.RuleID = "mutated again"
	listed[0].RefreshSummary.ReasonCodes[0] = "mutated again"
	assertSnapshot(t)

	loaded, ok := service.loadExistingRunRecord("run_snapshot_clone")
	if !ok {
		t.Fatal("expected loadExistingRunRecord result")
	}
	loaded.info.StepProviders["refresh.step1.collect"] = "mutated load"
	loaded.info.PendingPermissions[0].Decision.RuleID = "mutated load"
	loaded.info.RefreshSummary.ReasonCodes[0] = "mutated load"
	loaded.artifacts[0].Path = "mutated"
	assertSnapshot(t)
	artifacts, _ := service.GetRunArtifacts("run_snapshot_clone")
	if len(artifacts) != 1 || artifacts[0].Path != "reports/as-is/overview.md" {
		t.Fatalf("artifacts mutated through load helper: %+v", artifacts)
	}
}

func TestConcurrentRunSnapshotPollingDoesNotShareMutableState(t *testing.T) {
	t.Parallel()

	service := NewService()
	startedAt := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	if err := service.storeRun(runRecord{info: RunInfo{
		RunID:         "run_concurrent_snapshots",
		Pipeline:      string(PipelineRefresh),
		Status:        RunStatusRunning,
		StartedAt:     startedAt,
		StepProviders: map[string]string{"refresh.step1.collect": "qwen-code"},
		Warnings:      []string{"initial"},
		RefreshSummary: &RefreshSummary{
			ReasonCodes: []string{"initial"},
		},
	}}); err != nil {
		t.Fatalf("store initial run: %v", err)
	}

	var wg sync.WaitGroup
	for reader := 0; reader < 8; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for iteration := 0; iteration < 200; iteration++ {
				info, ok := service.GetRun("run_concurrent_snapshots")
				if !ok {
					return
				}
				info.StepProviders["refresh.step1.collect"] = "reader-mutated"
				if len(info.Warnings) > 0 {
					info.Warnings[0] = "reader-mutated"
				}
				if info.RefreshSummary != nil && len(info.RefreshSummary.ReasonCodes) > 0 {
					info.RefreshSummary.ReasonCodes[0] = "reader-mutated"
				}
				_ = service.ListRuns(0)
			}
		}()
	}
	for iteration := 0; iteration < 200; iteration++ {
		if err := service.storeRun(runRecord{info: RunInfo{
			RunID:         "run_concurrent_snapshots",
			Pipeline:      string(PipelineRefresh),
			Status:        RunStatusRunning,
			StartedAt:     startedAt,
			StepProviders: map[string]string{"refresh.step1.collect": "qwen-code"},
			Warnings:      []string{fmt.Sprintf("update-%d", iteration)},
			RefreshSummary: &RefreshSummary{
				ReasonCodes: []string{fmt.Sprintf("update-%d", iteration)},
			},
		}}); err != nil {
			t.Fatalf("store update %d: %v", iteration, err)
		}
	}
	wg.Wait()

	info, _ := service.GetRun("run_concurrent_snapshots")
	if info.StepProviders["refresh.step1.collect"] != "qwen-code" {
		t.Fatalf("registry map was mutated by reader: %+v", info.StepProviders)
	}
	if len(info.Warnings) != 1 || strings.HasPrefix(info.Warnings[0], "reader-") {
		t.Fatalf("registry warnings were mutated by reader: %v", info.Warnings)
	}
	if info.RefreshSummary == nil || strings.HasPrefix(info.RefreshSummary.ReasonCodes[0], "reader-") {
		t.Fatalf("registry refresh summary was mutated by reader: %+v", info.RefreshSummary)
	}
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
	waitForServiceQuiescent(t, service, 2*time.Second)
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
	if info.Status != RunStatusCanceled {
		t.Fatalf("expected shutdown-canceled run to be canceled, got %s", info.Status)
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
	if secondInfo.Status != RunStatusCanceled || secondInfo.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected pending run canceled/%s, got status=%s code=%q", runErrorCodeCanceled, secondInfo.Status, secondInfo.ErrorCode)
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

	if err := service.terminalizeActiveRunAfterUnexpectedExit(runID, context.Canceled, "run failed: context canceled"); err != nil {
		t.Fatalf("terminalize canceled run: %v", err)
	}

	history := readRunHistorySnapshot(t, ws.Path)
	item, ok := findHistoryItem(history, runID)
	if !ok {
		t.Fatalf("expected history item %q", runID)
	}
	if item.Status != RunStatusCanceled {
		t.Fatalf("expected canceled history status, got %s", item.Status)
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
	if runs := service.ListRuns(0); len(runs) != 0 {
		t.Fatalf("failed initial persistence must not publish a run: %+v", runs)
	}
	if diagnostics := service.HistoryDiagnostics(); len(diagnostics) == 0 || !strings.Contains(diagnostics[len(diagnostics)-1], "persist run history") {
		t.Fatalf("expected bounded history diagnostic, got %v", diagnostics)
	}
}

func TestHistoryPersistenceFailureKeepsLastPublishedRegistryAndHistory(t *testing.T) {
	ws := createWorkspace(t)
	service := NewService(WithHistoryWorkspace(ws))
	startedAt := time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)
	if err := service.storeRun(runRecord{info: RunInfo{
		RunID:         "run_transactional_history",
		Pipeline:      string(PipelineRefresh),
		Status:        RunStatusRunning,
		StartedAt:     startedAt,
		StepProviders: map[string]string{"refresh.step1.collect": "qwen-code"},
		RefreshSummary: &RefreshSummary{
			Mode:        "affected_only",
			ReasonCodes: []string{"source_changed"},
		},
	}}); err != nil {
		t.Fatalf("store initial run: %v", err)
	}
	beforeCurrent, err := os.ReadFile(filepath.Join(ws.Path, runHistoryPath))
	if err != nil {
		t.Fatalf("read current history: %v", err)
	}
	beforeLastGood, err := os.ReadFile(filepath.Join(ws.Path, runHistoryPath+".last-good"))
	if err != nil {
		t.Fatalf("read last-good history: %v", err)
	}

	originalWriter := service.historyWriteFile
	service.historyWriteFile = func(root workspace.Root, relPath string, content []byte) error {
		if relPath == runHistoryPath {
			return errors.New("injected current history failure")
		}
		return originalWriter(root, relPath, content)
	}

	finishedAt := startedAt.Add(time.Minute)
	update := RunInfo{
		RunID:         "run_transactional_history",
		Pipeline:      string(PipelineRefresh),
		Status:        RunStatusSucceeded,
		StartedAt:     startedAt,
		FinishedAt:    &finishedAt,
		StepProviders: map[string]string{"refresh.step1.collect": "claude-code"},
		RefreshSummary: &RefreshSummary{
			Mode:        "full",
			ReasonCodes: []string{"forced_full"},
		},
	}
	if err := service.storeRun(runRecord{info: update}); err == nil || !strings.Contains(err.Error(), "injected current history failure") {
		t.Fatalf("expected injected persistence failure, got %v", err)
	}

	published, ok := service.GetRun("run_transactional_history")
	if !ok {
		t.Fatal("expected previously published run")
	}
	if published.Status != RunStatusRunning || published.StepProviders["refresh.step1.collect"] != "qwen-code" || published.RefreshSummary.Mode != "affected_only" {
		t.Fatalf("failed transition leaked into registry: %+v", published)
	}
	afterCurrent, _ := os.ReadFile(filepath.Join(ws.Path, runHistoryPath))
	afterLastGood, _ := os.ReadFile(filepath.Join(ws.Path, runHistoryPath+".last-good"))
	if string(afterCurrent) != string(beforeCurrent) {
		t.Fatalf("current history changed after failed transition")
	}
	if string(afterLastGood) != string(beforeLastGood) {
		t.Fatalf("last-good history changed after failed transition")
	}
	if _, err := parseRunHistorySnapshot(afterCurrent); err != nil {
		t.Fatalf("current history became invalid: %v", err)
	}
	if _, err := parseRunHistorySnapshot(afterLastGood); err != nil {
		t.Fatalf("last-good history became invalid: %v", err)
	}

	restarted := NewService(WithHistoryWorkspace(ws))
	restarted.ReconcileStaleRunsAfterRestart()
	reconciled, ok := restarted.GetRun("run_transactional_history")
	if !ok || reconciled.Status != RunStatusFailed || reconciled.ErrorCode != runErrorCodeReconciledAfterRestart {
		t.Fatalf("restart did not reconcile last durable running truth: %+v", reconciled)
	}
}

func TestPendingReplacementIsOneRegistryTransaction(t *testing.T) {
	ws := createWorkspace(t)
	releaseRunner := make(chan struct{})
	runner := &countingBlockingRunner{release: releaseRunner}
	service := NewService(
		WithRunner(runner),
		WithHistoryWorkspace(ws),
	)
	activeRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start active run: %v", err)
	}
	waitForRunnerCalls(t, runner, 1, asyncRunnerStartTimeout)
	pendingRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
		Intent:         RunIntentQueue,
	})
	if err != nil {
		t.Fatalf("queue pending run: %v", err)
	}

	originalWriter := service.historyWriteFile
	failNextCurrent := true
	service.historyWriteFile = func(root workspace.Root, relPath string, content []byte) error {
		if relPath == runHistoryPath && failNextCurrent {
			failNextCurrent = false
			return errors.New("injected replacement transaction failure")
		}
		return originalWriter(root, relPath, content)
	}
	if _, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
		Intent:         RunIntentQueue,
	}); err == nil || !strings.Contains(err.Error(), "injected replacement transaction failure") {
		t.Fatalf("expected replacement transaction failure, got %v", err)
	}

	pending, ok := service.GetRun(pendingRunID)
	if !ok || pending.Status != RunStatusQueued || pending.SupersededByRunID != "" {
		t.Fatalf("failed replacement changed existing pending run: %+v", pending)
	}
	coordination := service.Coordination()
	if coordination.ActiveRunID != activeRunID || coordination.Pending == nil || coordination.Pending.RunID != pendingRunID {
		t.Fatalf("failed replacement changed coordination: %+v", coordination)
	}
	if runs := service.ListRuns(0); len(runs) != 2 {
		t.Fatalf("failed replacement published a partial registry: %+v", runs)
	}

	service.historyWriteFile = originalWriter
	close(releaseRunner)
	waitForRunTerminalInfo(t, service, activeRunID, 2*time.Second)
	waitForRunTerminalInfo(t, service, pendingRunID, 2*time.Second)
}

func TestLastGoodWriteFailurePublishesDurableCurrentAndDiagnostic(t *testing.T) {
	ws := createWorkspace(t)
	service := NewService(WithHistoryWorkspace(ws))
	originalWriter := service.historyWriteFile
	service.historyWriteFile = func(root workspace.Root, relPath string, content []byte) error {
		if relPath == runHistoryPath+".last-good" {
			return errors.New("injected last-good failure")
		}
		return originalWriter(root, relPath, content)
	}
	startedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	if err := service.storeRun(runRecord{info: RunInfo{
		RunID:     "run_current_committed",
		Pipeline:  string(PipelineInit),
		Status:    RunStatusSucceeded,
		StartedAt: startedAt,
	}}); err != nil {
		t.Fatalf("current history commit should succeed despite backup failure: %v", err)
	}
	if _, ok := service.GetRun("run_current_committed"); !ok {
		t.Fatal("durable current history was not published")
	}
	raw, err := os.ReadFile(filepath.Join(ws.Path, runHistoryPath))
	if err != nil {
		t.Fatalf("read durable current history: %v", err)
	}
	snapshot, err := parseRunHistorySnapshot(raw)
	if err != nil {
		t.Fatalf("parse durable current history: %v", err)
	}
	if _, ok := findHistoryItem(snapshot, "run_current_committed"); !ok {
		t.Fatalf("current history missing committed run: %+v", snapshot.Items)
	}
	diagnostics := service.HistoryDiagnostics()
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0], "last-good") {
		t.Fatalf("expected last-good diagnostic, got %v", diagnostics)
	}
}

func TestAsyncInitialProgressPersistenceFailureBecomesDurableFailure(t *testing.T) {
	ws := createWorkspace(t)
	service := NewService(WithHistoryWorkspace(ws))
	originalWriter := service.historyWriteFile
	var writerMu sync.Mutex
	currentWrites := 0
	service.historyWriteFile = func(root workspace.Root, relPath string, content []byte) error {
		if relPath == runHistoryPath {
			writerMu.Lock()
			currentWrites++
			writeNumber := currentWrites
			writerMu.Unlock()
			if writeNumber == 2 {
				return errors.New("injected initial running-state failure")
			}
		}
		return originalWriter(root, relPath, content)
	}

	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("queue async run: %v", err)
	}
	info := waitForRunTerminalInfo(t, service, runID, 2*time.Second)
	if info.Status != RunStatusFailed || info.ErrorCode != "internal_failure" {
		t.Fatalf("expected durable failed terminal state after transient persistence error, got %+v", info)
	}
	if !strings.Contains(info.Error, "persist initial run state") {
		t.Fatalf("expected persistence cause in terminal error, got %q", info.Error)
	}
	history := readRunHistorySnapshot(t, ws.Path)
	item, ok := findHistoryItem(history, runID)
	if !ok || item.Status != RunStatusFailed {
		t.Fatalf("expected durable failed history item, got %+v", item)
	}
	if coordination := service.Coordination(); coordination.ActiveRunID != "" {
		t.Fatalf("terminal persistence failure left active coordination: %+v", coordination)
	}
}

func TestHistoryDiagnosticsAreBoundedAndCopied(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.mu.Lock()
	for idx := 0; idx < historyDiagnosticsLimit+5; idx++ {
		service.addHistoryDiagnosticLocked(fmt.Sprintf("diagnostic-%02d", idx))
	}
	service.mu.Unlock()

	diagnostics := service.HistoryDiagnostics()
	if len(diagnostics) != historyDiagnosticsLimit {
		t.Fatalf("expected %d bounded diagnostics, got %d", historyDiagnosticsLimit, len(diagnostics))
	}
	if diagnostics[0] != "diagnostic-05" || diagnostics[len(diagnostics)-1] != "diagnostic-24" {
		t.Fatalf("unexpected retained diagnostic window: %v", diagnostics)
	}
	diagnostics[0] = "mutated"
	if current := service.HistoryDiagnostics(); current[0] != "diagnostic-05" {
		t.Fatalf("HistoryDiagnostics returned shared storage: %v", current)
	}
}

func TestRestartDoesNotReconcileCanceledRunAsFailed(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	finishedAt := time.Date(2026, 7, 26, 13, 0, 0, 0, time.UTC)
	service := NewService(WithHistoryWorkspace(ws))
	if err := service.storeRun(runRecord{info: RunInfo{
		RunID:      "run_canceled_before_restart",
		Pipeline:   string(PipelineRefresh),
		Status:     RunStatusCanceled,
		StartedAt:  finishedAt.Add(-time.Minute),
		FinishedAt: &finishedAt,
		ErrorCode:  runErrorCodeCanceled,
		Error:      "run canceled by request",
	}}); err != nil {
		t.Fatalf("store canceled run: %v", err)
	}

	restarted := NewService(WithHistoryWorkspace(ws))
	restarted.ReconcileStaleRunsAfterRestart()
	info, ok := restarted.GetRun("run_canceled_before_restart")
	if !ok {
		t.Fatal("expected canceled run after restart")
	}
	if info.Status != RunStatusCanceled || info.ErrorCode != runErrorCodeCanceled || info.FinishedAt == nil {
		t.Fatalf("restart changed canceled terminal truth: %+v", info)
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

func waitForServiceQuiescent(t *testing.T, service *Service, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !service.HasInFlightRun() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	coordination := service.Coordination()
	t.Fatalf(
		"service did not become quiescent before timeout; active_run_id=%q pending=%+v",
		coordination.ActiveRunID,
		coordination.Pending,
	)
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
