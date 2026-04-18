package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/testutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRunInitPipelineMaterializesExpectedArtifacts(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}
	if len(artifacts) == 0 {
		t.Fatalf("expected non-empty artifacts")
	}
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		key := artifact.Kind + "|" + artifact.Path
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate artifact entry detected: %s", key)
		}
		seen[key] = struct{}{}
	}

	for _, rel := range []string{
		"skills/subagents.yaml",
		"model/entities/svc.payments-service.yaml",
		"reports/as-is/overview.md",
		"reports/findings/findings.md",
		"reports/changelog",
	} {
		abs := filepath.Join(ws.Path, rel)
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("expected artifact %q: %v", rel, err)
		}
	}
}

func TestInitStep0FallsBackWhenWizardContractMissing(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}
	if !hasWarningPrefix(info.Warnings, "step0_wizard_contract_missing:") {
		t.Fatalf("expected step0 missing contract warning, got %+v", info.Warnings)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "charter/overview.md"))
	if err != nil {
		t.Fatalf("read charter overview: %v", err)
	}
	if !strings.Contains(string(content), "Generated baseline charter for ACP MVP.") {
		t.Fatalf("expected baseline overview fallback, got %q", string(content))
	}
}

func TestInitStep0FallsBackWhenWizardContractInvalid(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile(step0WizardContractPath, []byte("{invalid-json")); err != nil {
		t.Fatalf("write invalid step0 contract: %v", err)
	}

	service := NewService()
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}
	if !hasWarningPrefix(info.Warnings, "step0_wizard_contract_invalid:") {
		t.Fatalf("expected step0 invalid contract warning, got %+v", info.Warnings)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "charter/overview.md"))
	if err != nil {
		t.Fatalf("read charter overview: %v", err)
	}
	if !strings.Contains(string(content), "Generated baseline charter for ACP MVP.") {
		t.Fatalf("expected baseline overview fallback, got %q", string(content))
	}
}

func TestInitStep0UsesWizardContractDeterministically(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	step0Contract := `{
  "version": 1,
  "project_name": "Payments Platform",
  "scope": "payments, users, ci-cd",
  "nfr_priorities": ["availability", "traceability"],
  "rules": ["evidence-first findings", "no silent re-key"]
}
`
	if err := ws.WriteFile(step0WizardContractPath, []byte(step0Contract)); err != nil {
		t.Fatalf("write step0 contract: %v", err)
	}

	service := NewService()
	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("first init run failed: %v", err)
	}

	overviewPath := filepath.Join(ws.Path, "charter/overview.md")
	domainPath := filepath.Join(ws.Path, "charter/cards/domains/payments-service.md")
	teamPath := filepath.Join(ws.Path, "charter/cards/teams/platform.md")
	nfrPath := filepath.Join(ws.Path, "charter/nfr.yaml")
	rulesPath := filepath.Join(ws.Path, "charter/rules.yaml")

	firstOverview, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatalf("read first overview: %v", err)
	}
	firstDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read first domain card: %v", err)
	}
	firstTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read first team card: %v", err)
	}
	firstNFR, err := os.ReadFile(nfrPath)
	if err != nil {
		t.Fatalf("read first nfr: %v", err)
	}
	firstRules, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read first rules: %v", err)
	}

	if !strings.Contains(string(firstOverview), "- project_name: `Payments Platform`") {
		t.Fatalf("expected wizard project in overview, got %q", string(firstOverview))
	}
	if !strings.Contains(string(firstDomain), "- charter_scope: `payments, users, ci-cd`") {
		t.Fatalf("expected wizard scope in domain card, got %q", string(firstDomain))
	}
	if !strings.Contains(string(firstTeam), "- charter_project: `Payments Platform`") {
		t.Fatalf("expected wizard project in team card, got %q", string(firstTeam))
	}
	if !strings.Contains(string(firstNFR), "availability") || !strings.Contains(string(firstNFR), "traceability") {
		t.Fatalf("expected wizard nfr priorities in nfr.yaml, got %q", string(firstNFR))
	}
	if !strings.Contains(string(firstRules), "evidence-first findings") {
		t.Fatalf("expected wizard rules in rules.yaml, got %q", string(firstRules))
	}

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("second init run failed: %v", err)
	}

	secondOverview, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatalf("read second overview: %v", err)
	}
	secondDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read second domain card: %v", err)
	}
	secondTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read second team card: %v", err)
	}
	secondNFR, err := os.ReadFile(nfrPath)
	if err != nil {
		t.Fatalf("read second nfr: %v", err)
	}
	secondRules, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read second rules: %v", err)
	}

	if string(firstOverview) != string(secondOverview) {
		t.Fatalf("overview materialization from wizard contract is not deterministic")
	}
	if string(firstDomain) != string(secondDomain) {
		t.Fatalf("domain card materialization from wizard contract is not deterministic")
	}
	if string(firstTeam) != string(secondTeam) {
		t.Fatalf("team card materialization from wizard contract is not deterministic")
	}
	if string(firstNFR) != string(secondNFR) {
		t.Fatalf("nfr materialization from wizard contract is not deterministic")
	}
	if string(firstRules) != string(secondRules) {
		t.Fatalf("rules materialization from wizard contract is not deterministic")
	}
}

func TestStartAsyncRunRegistersAndCompletesRun(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()
	defer waitForAsyncDrain(t, service, 8*time.Second)

	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start async run: %v", err)
	}
	if runID == "" {
		t.Fatalf("expected run id")
	}

	info := waitForRunTerminalState(t, service, runID, 8*time.Second)
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected async run success, got status=%s error=%s", info.Status, info.Error)
	}
	if info.CurrentStep != "refresh.step4.proposals" {
		t.Fatalf("expected final current_step to point to last step, got %q", info.CurrentStep)
	}
}

func TestListRunsReturnsMostRecentFirstWithLimit(t *testing.T) {
	t.Parallel()

	service := NewService()
	baseTime := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:     "run_oldest",
			Pipeline:  string(PipelineInit),
			Status:    RunStatusSucceeded,
			StartedAt: baseTime,
		},
	})
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:     "run_middle",
			Pipeline:  string(PipelineRefresh),
			Status:    RunStatusSucceeded,
			StartedAt: baseTime.Add(1 * time.Minute),
		},
	})
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:     "run_newest",
			Pipeline:  string(PipelineRefresh),
			Status:    RunStatusRunning,
			StartedAt: baseTime.Add(2 * time.Minute),
		},
	})

	runs := service.ListRuns(2)
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs from limited list, got %d", len(runs))
	}
	if runs[0].RunID != "run_newest" || runs[1].RunID != "run_middle" {
		t.Fatalf("unexpected run order: %+v", runs)
	}
}

func TestRunHistoryPersistsAndLoadsAcrossServiceRestart(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
		}),
	)

	runInfo, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if runInfo.RunID == "" {
		t.Fatalf("expected non-empty run id")
	}

	historyContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/taskruns/run-history.json"))
	if err != nil {
		t.Fatalf("read run history file: %v", err)
	}
	var snapshot runHistorySnapshot
	if err := json.Unmarshal(historyContent, &snapshot); err != nil {
		t.Fatalf("decode run history snapshot: %v", err)
	}
	if snapshot.Version != runHistoryVersion {
		t.Fatalf("unexpected run history version: %d", snapshot.Version)
	}
	if len(snapshot.Items) == 0 {
		t.Fatalf("expected non-empty run history items")
	}

	restartedService := NewService(WithHistoryWorkspace(ws))
	reloadedInfo, ok := restartedService.GetRun(runInfo.RunID)
	if !ok {
		t.Fatalf("expected run %q to be loaded from persisted history", runInfo.RunID)
	}
	if reloadedInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status from persisted history, got %s", reloadedInfo.Status)
	}
}

func TestRunHistoryRetentionKeepsLast500Runs(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(WithHistoryWorkspace(ws))

	baseTime := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	totalRuns := runHistoryRetention + 5
	for idx := 0; idx < totalRuns; idx++ {
		service.storeRun(runRecord{
			info: RunInfo{
				RunID:     fmt.Sprintf("run_%04d", idx),
				Pipeline:  string(PipelineRefresh),
				Status:    RunStatusSucceeded,
				StartedAt: baseTime.Add(time.Duration(idx) * time.Second),
			},
		})
	}

	if got := runRegistrySize(service); got != runHistoryRetention {
		t.Fatalf("expected run registry retention %d, got %d", runHistoryRetention, got)
	}

	historyContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/taskruns/run-history.json"))
	if err != nil {
		t.Fatalf("read run history file: %v", err)
	}
	var snapshot runHistorySnapshot
	if err := json.Unmarshal(historyContent, &snapshot); err != nil {
		t.Fatalf("decode run history snapshot: %v", err)
	}
	if len(snapshot.Items) != runHistoryRetention {
		t.Fatalf("expected %d persisted runs, got %d", runHistoryRetention, len(snapshot.Items))
	}
	if snapshot.Items[0].RunID != "run_0005" {
		t.Fatalf("expected oldest retained run to be run_0005, got %q", snapshot.Items[0].RunID)
	}
	if snapshot.Items[len(snapshot.Items)-1].RunID != fmt.Sprintf("run_%04d", totalRuns-1) {
		t.Fatalf("expected latest retained run to be run_%04d, got %q", totalRuns-1, snapshot.Items[len(snapshot.Items)-1].RunID)
	}
}

func TestRunHistoryAsyncTransitionsQueuedRunningFinal(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(delayedRunner{delay: 220 * time.Millisecond}),
	)
	defer waitForAsyncDrain(t, service, 8*time.Second)

	run1, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start first async run: %v", err)
	}
	run2, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start second async run: %v", err)
	}

	if status := waitForRunHistoryStatus(t, ws, run2, 8*time.Second, RunStatusQueued); status != RunStatusQueued {
		t.Fatalf("expected queued status in persisted history, got %s", status)
	}
	if status := waitForRunHistoryStatus(t, ws, run2, 8*time.Second, RunStatusRunning); status != RunStatusRunning {
		t.Fatalf("expected running status in persisted history, got %s", status)
	}

	_ = waitForRunTerminalState(t, service, run1, 8*time.Second)
	final2 := waitForRunTerminalState(t, service, run2, 8*time.Second)
	if final2.Status != RunStatusSucceeded {
		t.Fatalf("expected second run success, got status=%s error=%q", final2.Status, final2.Error)
	}

	if status := waitForRunHistoryStatus(t, ws, run2, 8*time.Second, RunStatusSucceeded); status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status in persisted history, got %s", status)
	}
}

func TestStartAsyncRunFailsWhenWorkspaceLayoutCannotBeCreated(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	// Make `reports` a file to force EnsureLayout failure when creating nested dirs.
	if err := os.WriteFile(filepath.Join(ws.Path, "reports"), []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("write conflicting reports file: %v", err)
	}

	service := NewService()
	defer waitForAsyncDrain(t, service, 8*time.Second)
	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start async run: %v", err)
	}

	info := waitForRunTerminalState(t, service, runID, 8*time.Second)
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed run status, got %s", info.Status)
	}
	if info.Error == "" {
		t.Fatalf("expected non-empty error on failed run")
	}
}

func TestStartAsyncRunDebounceLastEventWins(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(delayedRunner{delay: 200 * time.Millisecond}),
		WithDebounceWindow(5*time.Minute),
	)
	defer waitForAsyncDrain(t, service, 8*time.Second)

	run1, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run1: %v", err)
	}
	run2, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run2: %v", err)
	}
	run3, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run3: %v", err)
	}
	if run2 == run3 {
		t.Fatalf("expected run3 to supersede run2 with a new run id")
	}

	info3 := waitForRunTerminalState(t, service, run3, 8*time.Second)
	if info3.Status != RunStatusSucceeded {
		t.Fatalf("expected run3 success, got %s (%s)", info3.Status, info3.Error)
	}

	info2 := waitForRunTerminalState(t, service, run2, 8*time.Second)
	if info2.Status != RunStatusFailed || info2.Error == "" {
		t.Fatalf("expected run2 superseded failure, got status=%s error=%q", info2.Status, info2.Error)
	}

	info1 := waitForRunTerminalState(t, service, run1, 8*time.Second)
	if info1.Status != RunStatusSucceeded {
		t.Fatalf("expected run1 success, got status=%s error=%q", info1.Status, info1.Error)
	}
}

func TestStartAsyncRunRejectsWhenPendingOutsideDebounceWindow(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(delayedRunner{delay: 220 * time.Millisecond}),
		WithDebounceWindow(10*time.Millisecond),
	)
	defer waitForAsyncDrain(t, service, 8*time.Second)

	run1, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start first run: %v", err)
	}
	run2, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start second run: %v", err)
	}
	beforeRejectRunCount := runRegistrySize(service)

	time.Sleep(25 * time.Millisecond)
	rejectedRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected debounce-window rejection")
	}
	if rejectedRunID != "" {
		t.Fatalf("expected empty run id on rejection, got %q", rejectedRunID)
	}
	if got := runRegistrySize(service); got != beforeRejectRunCount {
		t.Fatalf("expected run registry size to remain %d after rejection, got %d", beforeRejectRunCount, got)
	}
	assertRunRegistryContainsOnly(t, service, run1, run2)

	info1 := waitForRunTerminalState(t, service, run1, 8*time.Second)
	if info1.Status != RunStatusSucceeded {
		t.Fatalf("expected run1 success, got status=%s error=%q", info1.Status, info1.Error)
	}
	info2 := waitForRunTerminalState(t, service, run2, 8*time.Second)
	if info2.Status != RunStatusSucceeded {
		t.Fatalf("expected run2 success, got status=%s error=%q", info2.Status, info2.Error)
	}
}

func TestCancelRunPendingImmediateFailure(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	runner := &countingDelayedRunner{delay: 240 * time.Millisecond}
	service := NewService(
		WithRunner(runner),
		WithDebounceWindow(5*time.Minute),
	)
	defer waitForAsyncDrain(t, service, 8*time.Second)

	run1, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run1: %v", err)
	}
	run2, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run2: %v", err)
	}

	if err := service.CancelRun(run2); err != nil {
		t.Fatalf("cancel pending run2: %v", err)
	}

	info2 := waitForRunTerminalState(t, service, run2, 8*time.Second)
	if info2.Status != RunStatusFailed {
		t.Fatalf("expected run2 failed status, got %s", info2.Status)
	}
	if info2.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected run2 error_code %q, got %q", runErrorCodeCanceled, info2.ErrorCode)
	}
	if info2.CurrentStep != "" {
		t.Fatalf("expected run2 to never enter running step, got current_step %q", info2.CurrentStep)
	}
	if runner.callsForRun(run2) != 0 {
		t.Fatalf("expected no runtime invocations for canceled pending run, got %d", runner.callsForRun(run2))
	}

	info1 := waitForRunTerminalState(t, service, run1, 8*time.Second)
	if info1.Status != RunStatusSucceeded {
		t.Fatalf("expected run1 success, got status=%s error=%q", info1.Status, info1.Error)
	}
}

func TestCancelRunActiveCooperativeAndQueueContinues(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	runner := &countingDelayedRunner{delay: 260 * time.Millisecond}
	service := NewService(
		WithRunner(runner),
		WithDebounceWindow(5*time.Minute),
	)
	defer waitForAsyncDrain(t, service, 8*time.Second)

	run1, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run1: %v", err)
	}
	run2, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run2: %v", err)
	}

	testutil.WaitFor(t, 8*time.Second, testutil.WaitDescription("run1 did not become running"), func() (bool, error) {
		info, ok := service.GetRun(run1)
		if !ok {
			return false, nil
		}
		return info.Status == RunStatusRunning, nil
	})

	if err := service.CancelRun(run1); err != nil {
		t.Fatalf("cancel active run1: %v", err)
	}

	info1 := waitForRunTerminalState(t, service, run1, 8*time.Second)
	if info1.Status != RunStatusFailed {
		t.Fatalf("expected run1 failed status, got %s", info1.Status)
	}
	if info1.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected run1 error_code %q, got %q", runErrorCodeCanceled, info1.ErrorCode)
	}

	info2 := waitForRunTerminalState(t, service, run2, 8*time.Second)
	if info2.Status != RunStatusSucceeded {
		t.Fatalf("expected run2 succeeded after run1 cancel, got status=%s error=%q", info2.Status, info2.Error)
	}
	if runner.callsForRun(run2) == 0 {
		t.Fatalf("expected runtime invocations for run2 after canceled run1")
	}
}

func TestCancelRunActiveClassifiesRunnerKilledAsCanceled(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(WithRunner(cancelReturnsRunnerUnavailableRunner{}))
	defer waitForAsyncDrain(t, service, 8*time.Second)

	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	testutil.WaitFor(t, 8*time.Second, testutil.WaitDescription("run did not become running"), func() (bool, error) {
		info, ok := service.GetRun(runID)
		if !ok {
			return false, nil
		}
		return info.Status == RunStatusRunning, nil
	})

	if err := service.CancelRun(runID); err != nil {
		t.Fatalf("cancel active run: %v", err)
	}

	info := waitForRunTerminalState(t, service, runID, 8*time.Second)
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected error_code %q, got %q (error=%q)", runErrorCodeCanceled, info.ErrorCode, info.Error)
	}
	if !strings.Contains(info.Error, "run canceled by request") {
		t.Fatalf("expected cancel message in error, got %q", info.Error)
	}
}

func TestRunWithIDWorkspaceValidationPrefersRunCanceledWhenCancelWasRequested(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + filepath.Join(ws.Path, "repos", "payments-service") + `
    ref: definitely-missing-ref
  - name: users-service
    path: ` + filepath.Join(ws.Path, "repos", "users-service") + `
`
	if err := os.WriteFile(filepath.Join(ws.Path, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("rewrite manifest with invalid ref: %v", err)
	}
	ws, err := workspace.Open(ws.Path)
	if err != nil {
		t.Fatalf("reopen workspace: %v", err)
	}

	service := NewService()
	runID := "run-validation-canceled"
	service.cancelRequests[runID] = struct{}{}

	info, _, err := service.runWithID(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	}, runID)
	if err == nil {
		t.Fatalf("expected workspace validation failure")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != runErrorCodeCanceled {
		t.Fatalf("expected error_code %q, got %q", runErrorCodeCanceled, info.ErrorCode)
	}
	if !strings.Contains(info.Error, "run canceled by request") {
		t.Fatalf("expected cancel error message, got %q", info.Error)
	}
}

func TestRunAppliesRuntimeStepTimeoutFromWorkspaceManifest(t *testing.T) {
	clearRuntimeTimeoutEnvForTest(t)

	ws := createWorkspaceWithTimeouts(t, map[string]int{
		"step_timeout_sec": 1,
		"heartbeat_sec":    1,
	})
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(delayedRunner{delay: 2 * time.Second}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if !strings.Contains(strings.ToLower(info.Error), "timeout") {
		t.Fatalf("expected timeout context in error, got %q", info.Error)
	}
}

func TestRunEmitsRuntimeHeartbeatLogs(t *testing.T) {
	clearRuntimeTimeoutEnvForTest(t)

	ws := createWorkspaceWithTimeouts(t, map[string]int{
		"step_timeout_sec": 5,
		"heartbeat_sec":    1,
	})
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(delayedRunner{delay: 2200 * time.Millisecond}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	page, ok, err := service.GetRunLogs(info.RunID, 0, 500)
	if err != nil {
		t.Fatalf("get run logs: %v", err)
	}
	if !ok {
		t.Fatalf("expected run logs page")
	}
	foundHeartbeat := false
	for _, entry := range page.Items {
		if entry.Message == "runtime task heartbeat" {
			foundHeartbeat = true
			break
		}
	}
	if !foundHeartbeat {
		t.Fatalf("expected runtime task heartbeat log entry")
	}
}

func clearRuntimeTimeoutEnvForTest(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		acpruntime.RuntimeStepTimeoutEnv,
		acpruntime.RuntimeHeartbeatEnv,
		acpruntime.PipelineTimeoutEnv,
		acpruntime.PipelineKillGraceEnv,
		acpruntime.APIReadyTimeoutEnv,
		acpruntime.APIInitTimeoutEnv,
		acpruntime.UIInitPollTimeoutEnv,
		acpruntime.UICancelPollTimeoutEnv,
	} {
		t.Setenv(key, "")
	}
}

func TestNewServiceReconcilesStaleRunsAfterRestart(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return baseTime }),
	)

	service.storeRun(runRecord{
		info: RunInfo{
			RunID:       "run_queued_stale",
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusQueued,
			StartedAt:   baseTime.Add(-2 * time.Minute),
			CurrentStep: "",
		},
	})
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:       "run_running_stale",
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusRunning,
			StartedAt:   baseTime.Add(-3 * time.Minute),
			CurrentStep: "refresh.step1.collect",
		},
	})
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:       "run_succeeded",
			Pipeline:    string(PipelineInit),
			Status:      RunStatusSucceeded,
			StartedAt:   baseTime.Add(-5 * time.Minute),
			CurrentStep: "init.step4.proposals",
		},
	})

	reconciledAt := baseTime.Add(1 * time.Hour)
	restartedService := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return reconciledAt }),
	)

	queuedInfo, ok := restartedService.GetRun("run_queued_stale")
	if !ok {
		t.Fatalf("expected reconciled queued run in registry")
	}
	if queuedInfo.Status != RunStatusFailed {
		t.Fatalf("expected queued stale run failed after restart, got %s", queuedInfo.Status)
	}
	if queuedInfo.ErrorCode != runErrorCodeReconciledAfterRestart {
		t.Fatalf("expected queued stale error_code %q, got %q", runErrorCodeReconciledAfterRestart, queuedInfo.ErrorCode)
	}
	if queuedInfo.FinishedAt == nil || !queuedInfo.FinishedAt.UTC().Equal(reconciledAt) {
		t.Fatalf("expected queued stale finished_at=%s, got %+v", reconciledAt.Format(time.RFC3339), queuedInfo.FinishedAt)
	}

	runningInfo, ok := restartedService.GetRun("run_running_stale")
	if !ok {
		t.Fatalf("expected reconciled running run in registry")
	}
	if runningInfo.Status != RunStatusFailed {
		t.Fatalf("expected running stale run failed after restart, got %s", runningInfo.Status)
	}
	if runningInfo.ErrorCode != runErrorCodeReconciledAfterRestart {
		t.Fatalf("expected running stale error_code %q, got %q", runErrorCodeReconciledAfterRestart, runningInfo.ErrorCode)
	}

	succeededInfo, ok := restartedService.GetRun("run_succeeded")
	if !ok {
		t.Fatalf("expected succeeded run in registry")
	}
	if succeededInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected terminal run unchanged, got %s", succeededInfo.Status)
	}

	logPage, found, err := restartedService.GetRunLogs("run_running_stale", 0, 100)
	if err != nil {
		t.Fatalf("query run logs for reconciled run: %v", err)
	}
	if !found {
		t.Fatalf("expected logs to exist for reconciled run")
	}
	foundReconciliationEvent := false
	for _, item := range logPage.Items {
		if item.Message != "run reconciled after restart" {
			continue
		}
		code, _ := item.Fields["error_code"].(string)
		if code != runErrorCodeReconciledAfterRestart {
			t.Fatalf("expected reconciliation log error_code %q, got %q", runErrorCodeReconciledAfterRestart, code)
		}
		foundReconciliationEvent = true
		break
	}
	if !foundReconciliationEvent {
		t.Fatalf("expected reconciliation run-log event, got %+v", logPage.Items)
	}
}

func TestNewServiceAutoResumeLeavesQueuedAndArtifactlessRunningRunsReconciled(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return baseTime }),
	)

	service.storeRun(runRecord{
		info: RunInfo{
			RunID:       "run_queued_stale",
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusQueued,
			StartedAt:   baseTime.Add(-2 * time.Minute),
			CurrentStep: "",
		},
	})
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:       "run_running_no_artifacts",
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusRunning,
			StartedAt:   baseTime.Add(-3 * time.Minute),
			CurrentStep: "refresh.step1.collect",
		},
	})

	restartedService := NewService(
		WithHistoryWorkspace(ws),
		WithResumeStaleAsyncRuns(),
		WithRunner(&failOnRunRunner{}),
		WithClock(func() time.Time { return baseTime.Add(30 * time.Minute) }),
	)
	defer waitForAsyncDrain(t, restartedService, 8*time.Second)

	queuedInfo, ok := restartedService.GetRun("run_queued_stale")
	if !ok {
		t.Fatalf("expected queued run in registry")
	}
	if queuedInfo.Status != RunStatusFailed || queuedInfo.ErrorCode != runErrorCodeReconciledAfterRestart {
		t.Fatalf("expected queued run reconciled after restart, got %+v", queuedInfo)
	}

	runningInfo, ok := restartedService.GetRun("run_running_no_artifacts")
	if !ok {
		t.Fatalf("expected running run in registry")
	}
	if runningInfo.Status != RunStatusFailed || runningInfo.ErrorCode != runErrorCodeReconciledAfterRestart {
		t.Fatalf("expected artifactless running run reconciled after restart, got %+v", runningInfo)
	}
}

func TestNewServiceAutoResumeChoosesNewestResumableRunningRun(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time { return baseTime }),
	)

	service.storeRun(runRecord{
		info: RunInfo{
			RunID:       "run_running_older",
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusRunning,
			StartedAt:   baseTime.Add(-3 * time.Minute),
			CurrentStep: "refresh.step1.collect",
		},
	})
	service.storeRun(runRecord{
		info: RunInfo{
			RunID:       "run_running_newest",
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusRunning,
			StartedAt:   baseTime.Add(-1 * time.Minute),
			CurrentStep: "refresh.step1.collect",
		},
	})
	writeShardRecoveryMarker(t, ws, "run_running_older", "refresh.step1.collect")
	writeShardRecoveryMarker(t, ws, "run_running_newest", "refresh.step1.collect")

	resumeRunner := &failOnRunRunner{}
	restartedService := NewService(
		WithHistoryWorkspace(ws),
		WithResumeStaleAsyncRuns(),
		WithRunner(resumeRunner),
		WithClock(func() time.Time { return baseTime.Add(30 * time.Minute) }),
	)
	defer waitForAsyncDrain(t, restartedService, 8*time.Second)

	newestInfo := waitForRunTerminalState(t, restartedService, "run_running_newest", 8*time.Second)
	if newestInfo.ErrorCode == runErrorCodeReconciledAfterRestart {
		t.Fatalf("expected newest stale running run to be resumed, got %+v", newestInfo)
	}
	if resumeRunner.callCount() == 0 {
		t.Fatalf("expected resumed run to invoke runtime provider at least once")
	}

	olderInfo, ok := restartedService.GetRun("run_running_older")
	if !ok {
		t.Fatalf("expected older running run in registry")
	}
	if olderInfo.Status != RunStatusFailed || olderInfo.ErrorCode != runErrorCodeReconciledAfterRestart {
		t.Fatalf("expected older running run reconciled after restart, got %+v", olderInfo)
	}

	page, found, err := restartedService.GetRunLogs("run_running_newest", 0, 200)
	if err != nil {
		t.Fatalf("query logs for resumed run: %v", err)
	}
	if !found {
		t.Fatalf("expected logs for resumed run")
	}
	findRunLogByMessage(t, page.Items, "run resumed after restart")
}

func TestResumeStepForCurrentStepResumeCursor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		pipeline  Pipeline
		current   string
		wantStart string
	}{
		{
			name:      "init pre-runtime keeps current step",
			pipeline:  PipelineInit,
			current:   "init.step0.constitution",
			wantStart: "init.step0.constitution",
		},
		{
			name:      "init runtime step resumes from step1",
			pipeline:  PipelineInit,
			current:   "init.step1.collect",
			wantStart: "init.step1.collect",
		},
		{
			name:      "init downstream runtime resumes from step1",
			pipeline:  PipelineInit,
			current:   "init.step3.findings",
			wantStart: "init.step1.collect",
		},
		{
			name:      "refresh downstream runtime resumes from step1",
			pipeline:  PipelineRefresh,
			current:   "refresh.step4.proposals",
			wantStart: "refresh.step1.collect",
		},
		{
			name:      "unknown step yields empty cursor",
			pipeline:  PipelineRefresh,
			current:   "refresh.unknown",
			wantStart: "",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := resumeStepForCurrentStep(tc.pipeline, tc.current)
			if got != tc.wantStart {
				t.Fatalf("resumeStepForCurrentStep(%q, %q): got=%q want=%q", tc.pipeline, tc.current, got, tc.wantStart)
			}
		})
	}
}

func TestRuntimeFailureLogsIncludeSanitizedSnippets(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(runtimeFailureWithOutputRunner{}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected runtime failure")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	page, ok, logsErr := service.GetRunLogs(info.RunID, 0, 500)
	if logsErr != nil {
		t.Fatalf("query run logs: %v", logsErr)
	}
	if !ok {
		t.Fatalf("expected run logs for failed run")
	}
	entry := findRunLogByMessage(t, page.Items, "runtime task failed")
	stdoutSnippet, stdoutOk := entry.Fields["stdout_snippet"].(string)
	stderrSnippet, stderrOk := entry.Fields["stderr_snippet"].(string)
	if !stdoutOk || strings.TrimSpace(stdoutSnippet) == "" {
		t.Fatalf("expected stdout_snippet in runtime failure fields, got %+v", entry.Fields)
	}
	if !stderrOk || strings.TrimSpace(stderrSnippet) == "" {
		t.Fatalf("expected stderr_snippet in runtime failure fields, got %+v", entry.Fields)
	}
	if strings.Contains(stdoutSnippet, "\r") || strings.Contains(stderrSnippet, "\r") {
		t.Fatalf("expected snippets without carriage returns, got stdout=%q stderr=%q", stdoutSnippet, stderrSnippet)
	}
	if !strings.HasSuffix(stdoutSnippet, runtimeOutputSnippetSuffix) {
		t.Fatalf("expected stdout snippet to be truncated with suffix, got %q", stdoutSnippet)
	}
	if !strings.HasSuffix(stderrSnippet, runtimeOutputSnippetSuffix) {
		t.Fatalf("expected stderr snippet to be truncated with suffix, got %q", stderrSnippet)
	}
	stdoutRunes := len([]rune(stdoutSnippet))
	stderrRunes := len([]rune(stderrSnippet))
	limitRunes := runtimeOutputSnippetLimitRunes + len([]rune(runtimeOutputSnippetSuffix))
	if stdoutRunes > limitRunes {
		t.Fatalf("stdout snippet exceeds deterministic limit: got=%d limit=%d", stdoutRunes, limitRunes)
	}
	if stderrRunes > limitRunes {
		t.Fatalf("stderr snippet exceeds deterministic limit: got=%d limit=%d", stderrRunes, limitRunes)
	}
}

func TestRuntimeParseFailureLogsIncludeSanitizedSnippets(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(runtimeParseFailureWithOutputRunner{}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected parse failure")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	page, ok, logsErr := service.GetRunLogs(info.RunID, 0, 500)
	if logsErr != nil {
		t.Fatalf("query run logs: %v", logsErr)
	}
	if !ok {
		t.Fatalf("expected run logs for failed run")
	}
	entry := findRunLogByMessage(t, page.Items, "runtime task parse failed")
	stdoutSnippet, stdoutOk := entry.Fields["stdout_snippet"].(string)
	stderrSnippet, stderrOk := entry.Fields["stderr_snippet"].(string)
	if !stdoutOk || strings.TrimSpace(stdoutSnippet) == "" {
		t.Fatalf("expected stdout_snippet in parse failure fields, got %+v", entry.Fields)
	}
	if !stderrOk || strings.TrimSpace(stderrSnippet) == "" {
		t.Fatalf("expected stderr_snippet in parse failure fields, got %+v", entry.Fields)
	}
	if strings.Contains(stdoutSnippet, "\r") || strings.Contains(stderrSnippet, "\r") {
		t.Fatalf("expected snippets without carriage returns, got stdout=%q stderr=%q", stdoutSnippet, stderrSnippet)
	}
	stdoutRunes := len([]rune(stdoutSnippet))
	stderrRunes := len([]rune(stderrSnippet))
	limitRunes := runtimeOutputSnippetLimitRunes + len([]rune(runtimeOutputSnippetSuffix))
	if stdoutRunes > limitRunes {
		t.Fatalf("stdout snippet exceeds deterministic limit: got=%d limit=%d", stdoutRunes, limitRunes)
	}
	if stderrRunes > limitRunes {
		t.Fatalf("stderr snippet exceeds deterministic limit: got=%d limit=%d", stderrRunes, limitRunes)
	}
}

func TestRunInitPipelineMaterializesDocArtifactMetadata(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docArtifactRunner{}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init with doc artifact runner: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "reports/taskruns/doc-artifacts.md"))
	if err != nil {
		t.Fatalf("read doc-artifacts report: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Doc Artifact Metadata") {
		t.Fatalf("expected metadata title in doc-artifacts report, got %q", text)
	}
	if !strings.Contains(text, "imports.architecture.notes") {
		t.Fatalf("expected doc artifact id in report, got %q", text)
	}
	if !strings.Contains(text, "docs/imports/architecture-notes.md") {
		t.Fatalf("expected doc artifact path in report, got %q", text)
	}
}

func TestInitStep1EnrichesCanonicalCardsDeterministically(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("first init run failed: %v", err)
	}

	domainPath := filepath.Join(ws.Path, "charter/cards/domains/payments-service.md")
	teamPath := filepath.Join(ws.Path, "charter/cards/teams/platform.md")

	firstDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read first domain card: %v", err)
	}
	firstTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read first team card: %v", err)
	}
	if strings.Contains(string(firstDomain), "## Derived (ACP Step1)") {
		t.Fatalf("did not expect runtime-derived section in domain card, got %q", string(firstDomain))
	}
	if strings.Contains(string(firstTeam), "## Derived (ACP Step1)") {
		t.Fatalf("did not expect runtime-derived section in team card, got %q", string(firstTeam))
	}

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("second init run failed: %v", err)
	}

	secondDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read second domain card: %v", err)
	}
	secondTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read second team card: %v", err)
	}
	if string(firstDomain) != string(secondDomain) {
		t.Fatalf("domain card enrichment is not deterministic")
	}
	if string(firstTeam) != string(secondTeam) {
		t.Fatalf("team card enrichment is not deterministic")
	}
}

func TestInitStep1ExecutesPerCanonicalDomainAndMaterializesDomainTaskruns(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	runner := &trackingRunner{}
	service := NewService(
		WithRunner(runner),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}

	domainIDs, err := loadCanonicalDomainIDs(ws)
	if err != nil {
		t.Fatalf("load canonical domain ids: %v", err)
	}
	if len(domainIDs) == 0 {
		t.Fatalf("expected canonical domain cards to exist after init")
	}

	collectTasks := runner.tasksForStep("init.step1.collect")
	if len(collectTasks) != len(domainIDs) {
		t.Fatalf("expected %d step1 runtime executions, got %d", len(domainIDs), len(collectTasks))
	}

	expectedScopes := collectRepoScopes(ws.Manifest.Repos)
	seenScopes := make([]string, 0, len(collectTasks))
	for _, task := range collectTasks {
		if len(task.RepoScopes) != 1 {
			t.Fatalf("expected exactly one repo scope per domain task, got %v", task.RepoScopes)
		}
		seenScopes = append(seenScopes, task.RepoScopes[0])
	}
	sort.Strings(seenScopes)
	if len(seenScopes) != len(expectedScopes) {
		t.Fatalf("unexpected scope count: got %d want %d", len(seenScopes), len(expectedScopes))
	}
	for idx := range seenScopes {
		if seenScopes[idx] != expectedScopes[idx] {
			t.Fatalf("unexpected per-domain scopes: got %v want %v", seenScopes, expectedScopes)
		}
	}

	for _, domainID := range domainIDs {
		taskrunPath := filepath.Join(
			ws.Path,
			"reports",
			"taskruns",
			fmt.Sprintf("%s-init-step1-collect-domain-%s.json", info.RunID, sanitizeDomainArtifactSlug(domainID)),
		)
		taskrun, readErr := os.ReadFile(taskrunPath)
		if readErr != nil {
			t.Fatalf("read per-domain taskrun %s: %v", taskrunPath, readErr)
		}
		var payload struct {
			Meta struct {
				StepID string `json:"step_id"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(taskrun, &payload); err != nil {
			t.Fatalf("decode per-domain taskrun %s: %v", taskrunPath, err)
		}
		if payload.Meta.StepID != "init.step1.collect" {
			t.Fatalf("expected per-domain taskrun step id init.step1.collect, got %q", payload.Meta.StepID)
		}

		domainOutputPath := filepath.Join(ws.Path, "reports", "agent-outputs", "domains", fmt.Sprintf("%s.md", domainID))
		if _, statErr := os.Stat(domainOutputPath); statErr != nil {
			t.Fatalf("expected domain output %s: %v", domainOutputPath, statErr)
		}
	}
}

func TestArchitectSummaryIsDeterministicAcrossRuns(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("first init run failed: %v", err)
	}
	summaryPath := filepath.Join(ws.Path, "reports", "agent-outputs", "architect", "summary.md")
	firstSummary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read first architect summary: %v", err)
	}

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("second init run failed: %v", err)
	}
	secondSummary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read second architect summary: %v", err)
	}

	if string(firstSummary) != string(secondSummary) {
		t.Fatalf("architect summary is not deterministic across runs")
	}
	text := string(firstSummary)
	if strings.Contains(text, "run_") {
		t.Fatalf("architect summary must not contain run id markers: %q", text)
	}
	paymentsIdx := strings.Index(text, "`payments-service`")
	usersIdx := strings.Index(text, "`users-service`")
	if paymentsIdx < 0 || usersIdx < 0 {
		t.Fatalf("expected architect summary to include canonical domains, got %q", text)
	}
	if paymentsIdx > usersIdx {
		t.Fatalf("expected sorted domain order in architect summary, got %q", text)
	}
}

func TestRefreshStep1MissingCanonicalDomainsWritesQuestionWithoutAutoCreate(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(WithRunner(step3ParseFailureRunner{}))

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected refresh run to fail at step3")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	if !strings.Contains(string(questionsReport), "q.domains.missing-canonical-cards") {
		t.Fatalf("expected missing-canonical-domains question in coverage report, got %q", string(questionsReport))
	}
	if !strings.Contains(string(questionsReport), "q.teams.missing-canonical-cards") {
		t.Fatalf("expected missing-canonical-teams question in coverage report, got %q", string(questionsReport))
	}

	matches, err := filepath.Glob(filepath.Join(ws.Path, "charter/cards/domains", "*.md"))
	if err != nil {
		t.Fatalf("glob canonical domain cards: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("step1 must not auto-create canonical domain cards, found: %v", matches)
	}
	teamMatches, err := filepath.Glob(filepath.Join(ws.Path, "charter/cards/teams", "*.md"))
	if err != nil {
		t.Fatalf("glob canonical team cards: %v", err)
	}
	if len(teamMatches) != 0 {
		t.Fatalf("step1 must not auto-create canonical team cards, found: %v", teamMatches)
	}
}

func TestRefreshStep1MissingRepoScopeWritesQuestion(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(step3ParseFailureRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected refresh run to fail at step3")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	if !strings.Contains(string(questionsReport), "q.domain.billing.missing-repo-scope") {
		t.Fatalf("expected missing repo scope question in coverage report, got %q", string(questionsReport))
	}
}

func TestRefreshStep1UsesDeclaredRepoScopeForMonolithDomain(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/shipping.md", []byte("# Domain: Shipping\n\n- id: `shipping`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write shipping domain card: %v", err)
	}

	runner := &trackingRunner{}
	service := NewService(WithRunner(runner))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("expected refresh run to succeed, got %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	collectTasks := runner.tasksForStep("refresh.step1.collect")
	if len(collectTasks) != 2 {
		t.Fatalf("expected two domain collect tasks, got %d", len(collectTasks))
	}
	for _, task := range collectTasks {
		if len(task.RepoScopes) != 1 || task.RepoScopes[0] != "orders-monolith" {
			t.Fatalf("expected monolith repo scope in collect task, got %+v", task.RepoScopes)
		}
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	text := string(questionsReport)
	if strings.Contains(text, "q.domain.billing.missing-repo-scope") {
		t.Fatalf("did not expect missing repo scope question, got %q", text)
	}
	if strings.Contains(text, "q.domain.billing.unknown-repo-scope") {
		t.Fatalf("did not expect unknown repo scope question, got %q", text)
	}
}

func TestRefreshStep1UnknownDeclaredRepoScopeWritesQuestion(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `unknown-scope`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(step3ParseFailureRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected refresh run to fail at step3")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	text := string(questionsReport)
	if !strings.Contains(text, "q.domain.billing.unknown-repo-scope") {
		t.Fatalf("expected unknown repo scope question, got %q", text)
	}
	if strings.Contains(text, "q.domain.billing.missing-repo-scope") {
		t.Fatalf("did not expect missing repo scope question when repo_scope is declared, got %q", text)
	}
}

func TestResolveRepoScopeForDomainCardContentDeclaredUnknownFallsBackToSlug(t *testing.T) {
	t.Parallel()

	repos := []workspace.RepoSource{
		{Name: "billing", Path: "./repos/billing"},
	}
	content := "# Domain: Billing\n\n- id: `billing`\n- repo_scope: `unknown-scope`\n"

	resolution := resolveRepoScopeForDomainCardContent("billing", content, repos)
	if !resolution.HasDeclaredRepoScope {
		t.Fatalf("expected declared repo_scope to be detected")
	}
	if resolution.DeclaredRepoScopeKnown {
		t.Fatalf("expected declared repo_scope to be unknown")
	}
	if resolution.RepoScope != "billing" {
		t.Fatalf("expected slug fallback repo_scope billing, got %q", resolution.RepoScope)
	}
}

func TestRefreshStep1RepoScopeResolverIsConsistentForRuntimeAndEnrich(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/payments.md", []byte("# Domain: Payments\n\n- id: `payments`\n- repo_scope: `users-service`\n")); err != nil {
		t.Fatalf("write payments domain card: %v", err)
	}

	runner := &trackingRunner{}
	service := NewService(WithRunner(runner))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	collectTasks := runner.tasksForStep("refresh.step1.collect")
	if len(collectTasks) != 1 {
		t.Fatalf("expected one domain collect task, got %d", len(collectTasks))
	}
	if len(collectTasks[0].RepoScopes) != 1 || collectTasks[0].RepoScopes[0] != "users-service" {
		t.Fatalf("expected runtime step1 to use declared repo_scope users-service, got %+v", collectTasks[0].RepoScopes)
	}

	cardBytes, err := os.ReadFile(filepath.Join(ws.Path, "charter/cards/domains/payments.md"))
	if err != nil {
		t.Fatalf("read enriched domain card: %v", err)
	}
	cardText := string(cardBytes)
	if strings.Contains(cardText, "## Derived (ACP Step1)") {
		t.Fatalf("did not expect runtime-derived section in domain card, got %q", cardText)
	}
	if !strings.Contains(cardText, "- repo_scope: `users-service`") {
		t.Fatalf("expected canonical card to keep declared repo_scope users-service, got %q", cardText)
	}
}

func TestRefreshStep1DomainCardIDMismatchAddsQuestionAndKeepsFilenameDomainID(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `payments`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(step3ParseFailureRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected refresh run to fail at step3")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	if !strings.Contains(string(questionsReport), "q.domain.billing.id-mismatch") {
		t.Fatalf("expected id mismatch question in coverage report, got %q", string(questionsReport))
	}

	if _, err := os.Stat(filepath.Join(ws.Path, "reports/agent-outputs/domains/billing.md")); err != nil {
		t.Fatalf("expected domain output to keep filename-based domain id, stat failed: %v", err)
	}
}

func TestRefreshIncludesAllDomainRepoScopesAndOmitsRepoSelectionSummaryArtifact(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	backendA := filepath.Join(root, "repos", "payments-service")
	backendB := filepath.Join(root, "repos", "users-service")
	frontend := filepath.Join(root, "repos", "web-frontend")
	for _, repoPath := range []string{backendA, backendB, frontend} {
		if err := os.MkdirAll(repoPath, 0o755); err != nil {
			t.Fatalf("create repo path %q: %v", repoPath, err)
		}
		if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# repo\n"), 0o644); err != nil {
			t.Fatalf("write readme %q: %v", repoPath, err)
		}
	}
	manifest := workspace.Manifest{
		Version: 1,
		Repos: []workspace.RepoSource{
			{Name: "payments-service", Path: backendA, Analysis: &workspace.RepoAnalysisConfig{Role: workspace.RepoRoleBackend}},
			{Name: "users-service", Path: backendB, Analysis: &workspace.RepoAnalysisConfig{Role: workspace.RepoRoleBackend}},
			{Name: "web-frontend", Path: frontend, Analysis: &workspace.RepoAnalysisConfig{Role: workspace.RepoRoleFrontend}},
		},
	}
	manifestRaw, err := workspace.RenderManifest(manifest)
	if err != nil {
		t.Fatalf("render manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, workspace.ManifestFileName), manifestRaw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/payments.md", []byte("# Domain: Payments\n\n- id: `payments`\n- repo_scope: `payments-service`\n")); err != nil {
		t.Fatalf("write payments domain card: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/users.md", []byte("# Domain: Users\n\n- id: `users`\n- repo_scope: `users-service`\n")); err != nil {
		t.Fatalf("write users domain card: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/web.md", []byte("# Domain: Web\n\n- id: `web`\n- repo_scope: `web-frontend`\n")); err != nil {
		t.Fatalf("write web domain card: %v", err)
	}

	runner := &trackingRunner{}
	service := NewService(WithRunner(runner))
	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	collectTasks := runner.tasksForStep("refresh.step1.collect")
	if len(collectTasks) != 3 {
		t.Fatalf("expected three domain collect tasks, got %d", len(collectTasks))
	}
	foundFrontendCollect := false
	for _, task := range collectTasks {
		if containsString(task.RepoScopes, "web-frontend") {
			foundFrontendCollect = true
		}
	}
	if !foundFrontendCollect {
		t.Fatalf("expected frontend scope to remain included in step1 collect tasks")
	}
	step3Tasks := runner.tasksForStep("refresh.step3.findings")
	if len(step3Tasks) == 0 {
		t.Fatalf("expected step3 findings tasks")
	}
	foundFrontendFindings := false
	for _, task := range step3Tasks {
		if containsString(task.RepoScopes, "web-frontend") {
			foundFrontendFindings = true
		}
	}
	if !foundFrontendFindings {
		t.Fatalf("expected frontend scope to remain included in step3 tasks")
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	if strings.Contains(string(questionsReport), "repo-scope-excluded-by-selection") {
		t.Fatalf("did not expect legacy repo-selection exclusion question, got %q", string(questionsReport))
	}

	repoSelectionArtifactPath := "reports/taskruns/" + info.RunID + "-repo-selection-summary.json"
	if _, err := os.Stat(filepath.Join(ws.Path, repoSelectionArtifactPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected repo selection summary artifact to be absent, stat err=%v", err)
	}
	for _, artifact := range artifacts {
		if artifact.Path == repoSelectionArtifactPath {
			t.Fatalf("did not expect repo selection summary artifact %q in run artifacts", repoSelectionArtifactPath)
		}
	}
}

func TestRefreshUnknownDeclaredRepoScopeFallsBackToResolvedRepoScopeWithoutSelectionSkip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "billing")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create billing repo path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# billing\n"), 0o644); err != nil {
		t.Fatalf("write billing readme: %v", err)
	}

	manifest := workspace.Manifest{
		Version: 1,
		Repos: []workspace.RepoSource{
			{Name: "billing", Path: repoPath, Analysis: &workspace.RepoAnalysisConfig{Role: workspace.RepoRoleFrontend}},
		},
	}
	manifestRaw, err := workspace.RenderManifest(manifest)
	if err != nil {
		t.Fatalf("render manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, workspace.ManifestFileName), manifestRaw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `unknown-scope`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	runner := &trackingRunner{}
	service := NewService(WithRunner(runner))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	text := string(questionsReport)
	if strings.Contains(text, "repo-scope-excluded-by-selection") {
		t.Fatalf("did not expect excluded-by-selection question, got %q", text)
	}
	if !strings.Contains(text, "q.domain.billing.unknown-repo-scope") {
		t.Fatalf("expected unknown-repo-scope question, got %q", text)
	}
	collectTasks := runner.tasksForStep("refresh.step1.collect")
	if len(collectTasks) != 1 {
		t.Fatalf("expected one collect task, got %d", len(collectTasks))
	}
	if !containsString(collectTasks[0].RepoScopes, "billing") {
		t.Fatalf("expected fallback repo scope billing in collect task, got %+v", collectTasks[0].RepoScopes)
	}
}

func TestSemanticGuardDropsRuntimeProviderEntityInRefreshStep1Collect(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(refreshCollectNoiseRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step1.collect: semantic_guard: dropped refresh.step1.collect entity types") {
		t.Fatalf("expected semantic guard warning in run warnings, got %#v", info.Warnings)
	}

	entities, err := model.NewStore(ws).ListEntities()
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}
	for _, entity := range entities {
		if entity.Type == "runtime_provider" {
			t.Fatalf("runtime_provider entity must be filtered by semantic guard, found %q", entity.ID)
		}
	}
}

func TestSemanticGuardDropsOffTopicArtifactsInRefreshStep1Collect(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(refreshCollectOffTopicRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step1.collect: semantic_guard: dropped refresh.step1.collect off-topic artifacts") {
		t.Fatalf("expected off-topic semantic guard warning in run warnings, got %#v", info.Warnings)
	}

	taskrunsDir := filepath.Join(ws.Path, "reports", "taskruns")
	step1Taskruns, err := filepath.Glob(filepath.Join(taskrunsDir, "*-refresh-step1-collect-domain-*.json"))
	if err != nil {
		t.Fatalf("glob refresh step1 taskruns: %v", err)
	}
	if len(step1Taskruns) == 0 {
		t.Fatalf("expected refresh step1 taskrun files")
	}
	latest := step1Taskruns[len(step1Taskruns)-1]
	raw, err := os.ReadFile(latest)
	if err != nil {
		t.Fatalf("read taskrun: %v", err)
	}
	var payload contracts.TaskResult
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal taskrun payload: %v", err)
	}
	for _, op := range payload.Changeset {
		if op.Op != "upsert_entity" || op.Entity == nil {
			continue
		}
		entityText := strings.ToLower(strings.Join([]string{op.Entity.ID, op.Entity.Type, op.Entity.Name}, " "))
		if strings.Contains(entityText, "chinabidding") || strings.Contains(entityText, "bidding") {
			t.Fatalf("off-topic entity should be filtered from normalized taskrun, got %q", entityText)
		}
	}
	for _, question := range payload.Questions {
		if strings.Contains(strings.ToLower(question.Text), "bidding") {
			t.Fatalf("off-topic question should be filtered from normalized taskrun, got %q", question.Text)
		}
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	if strings.Contains(strings.ToLower(string(questionsReport)), "bidding") {
		t.Fatalf("off-topic question should be filtered from coverage questions report, got %q", string(questionsReport))
	}
}

func TestSemanticGuardMarksCriticalOffTopicDriftInRefreshStep1Collect(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(refreshCollectCriticalOffTopicRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step1.collect: semantic_guard: critical_off_topic_drift in refresh.step1.collect") {
		t.Fatalf("expected critical off-topic drift warning in run warnings, got %#v", info.Warnings)
	}

	taskrunsDir := filepath.Join(ws.Path, "reports", "taskruns")
	step1Taskruns, err := filepath.Glob(filepath.Join(taskrunsDir, "*-refresh-step1-collect-domain-*.json"))
	if err != nil {
		t.Fatalf("glob refresh step1 taskruns: %v", err)
	}
	if len(step1Taskruns) == 0 {
		t.Fatalf("expected refresh step1 taskrun files")
	}
	latest := step1Taskruns[len(step1Taskruns)-1]
	raw, err := os.ReadFile(latest)
	if err != nil {
		t.Fatalf("read taskrun: %v", err)
	}
	var payload contracts.TaskResult
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal taskrun payload: %v", err)
	}
	foundCriticalWarning := false
	for _, warning := range payload.Warnings {
		if strings.Contains(warning, "semantic_guard: critical_off_topic_drift in refresh.step1.collect") {
			foundCriticalWarning = true
			break
		}
	}
	if !foundCriticalWarning {
		t.Fatalf("expected critical off-topic warning in normalized step1 taskrun payload, got %#v", payload.Warnings)
	}
}

func TestSemanticGuardAddsFallbackFindingForOwnerGapInRefreshStep3(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(refreshMissingFindingsRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step3.findings: semantic_guard: added fallback owner-mapping finding") {
		t.Fatalf("expected fallback-finding warning in run warnings, got %#v", info.Warnings)
	}

	findingsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/findings/findings.md"))
	if err != nil {
		t.Fatalf("read findings report: %v", err)
	}
	text := string(findingsReport)
	if strings.Contains(text, "No findings reported.") {
		t.Fatalf("expected fallback finding to be materialized, got %q", text)
	}
	if !strings.Contains(text, "Missing owner mapping") {
		t.Fatalf("expected owner mapping finding in report, got %q", text)
	}
}

func TestSemanticGuardAddsGenericFallbackFindingWhenNoServiceCandidate(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(refreshNoServiceMissingFindingsRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step3.findings: semantic_guard: added fallback owner-mapping finding") {
		t.Fatalf("expected fallback-finding warning in run warnings, got %#v", info.Warnings)
	}

	findingsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/findings/findings.md"))
	if err != nil {
		t.Fatalf("read findings report: %v", err)
	}
	text := string(findingsReport)
	if !strings.Contains(text, "owner mappings are unresolved for repo scope") {
		t.Fatalf("expected generic fallback finding description in report, got %q", text)
	}
	if !strings.Contains(text, "`scope.orders-monolith`") {
		t.Fatalf("expected generic fallback related id in report, got %q", text)
	}
}

func TestSemanticGuardAddsFallbackCrossRepoEdgeForMultiScopeRefreshStep3(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/payments.md", []byte("# Domain: Payments\n\n- id: `payments`\n- repo_scope: `payments-service`\n")); err != nil {
		t.Fatalf("write payments domain card: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/users.md", []byte("# Domain: Users\n\n- id: `users`\n- repo_scope: `users-service`\n")); err != nil {
		t.Fatalf("write users domain card: %v", err)
	}

	service := NewService(WithRunner(refreshMissingFindingsRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step3.findings: semantic_guard: added fallback cross-repo edge") {
		t.Fatalf("expected fallback cross-repo edge warning in run warnings, got %#v", info.Warnings)
	}

	step3TaskrunCandidates, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-refresh-step3-findings*.json"))
	if err != nil {
		t.Fatalf("glob step3 taskruns: %v", err)
	}
	step3Taskruns := make([]string, 0, len(step3TaskrunCandidates))
	for _, candidate := range step3TaskrunCandidates {
		if strings.Contains(candidate, "shard-summary") {
			continue
		}
		step3Taskruns = append(step3Taskruns, candidate)
	}
	sort.Strings(step3Taskruns)
	if len(step3Taskruns) == 0 {
		t.Fatalf("expected refresh step3 taskrun file")
	}
	raw, err := os.ReadFile(step3Taskruns[len(step3Taskruns)-1])
	if err != nil {
		t.Fatalf("read step3 taskrun: %v", err)
	}
	var payload contracts.TaskResult
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal step3 taskrun payload: %v", err)
	}
	foundEdge := false
	for _, op := range payload.Changeset {
		if op.Op == "upsert_edge" && op.Edge != nil {
			foundEdge = true
			if strings.TrimSpace(op.Edge.From) == strings.TrimSpace(op.Edge.To) {
				t.Fatalf("expected cross-repo edge with different endpoints, got %+v", op.Edge)
			}
			break
		}
	}
	if !foundEdge {
		t.Fatalf("expected fallback cross-repo upsert_edge in step3 taskrun, got %#v", payload.Changeset)
	}
}

func TestSemanticGuardRemovesInvalidEvidenceAndDowngradesObservation(t *testing.T) {
	t.Parallel()

	ws := createMonolithWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n- repo_scope: `orders-monolith`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(refreshInvalidEvidenceRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step1.collect: semantic_guard: removed invalid evidence paths count=") {
		t.Fatalf("expected invalid-evidence warning in run warnings, got %#v", info.Warnings)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step1.collect: semantic_guard: downgraded observation provenance to inference count=") {
		t.Fatalf("expected observation downgrade warning in run warnings, got %#v", info.Warnings)
	}

	step1Taskruns, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-refresh-step1-collect-domain-*.json"))
	if err != nil {
		t.Fatalf("glob step1 taskruns: %v", err)
	}
	if len(step1Taskruns) == 0 {
		t.Fatalf("expected refresh step1 taskrun files")
	}
	raw, err := os.ReadFile(step1Taskruns[len(step1Taskruns)-1])
	if err != nil {
		t.Fatalf("read step1 taskrun: %v", err)
	}
	var payload contracts.TaskResult
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal step1 taskrun payload: %v", err)
	}

	for _, op := range payload.Changeset {
		if op.Op != "upsert_entity" || op.Entity == nil {
			continue
		}
		if strings.TrimSpace(op.Entity.Provenance.Kind) != "inference" {
			t.Fatalf("expected entity provenance to be downgraded to inference, got %q", op.Entity.Provenance.Kind)
		}
		for _, evidence := range op.Entity.Provenance.Evidence {
			if evidence.Path == "/" || evidence.Path == "." {
				t.Fatalf("expected invalid evidence paths to be removed, got %+v", op.Entity.Provenance.Evidence)
			}
		}
		return
	}
	t.Fatalf("expected at least one upsert_entity operation in step1 taskrun")
}

func TestSemanticGuardNormalizesMultiRepoMissingEdgeAndInvalidEvidence(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/payments.md", []byte("# Domain: Payments\n\n- id: `payments`\n- repo_scope: `payments-service`\n")); err != nil {
		t.Fatalf("write payments domain card: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/users.md", []byte("# Domain: Users\n\n- id: `users`\n- repo_scope: `users-service`\n")); err != nil {
		t.Fatalf("write users domain card: %v", err)
	}

	service := NewService(WithRunner(refreshMultiScopeNoEdgeInvalidEvidenceRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step1.collect: semantic_guard: removed invalid evidence paths count=") {
		t.Fatalf("expected invalid-evidence warning in run warnings, got %#v", info.Warnings)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step1.collect: semantic_guard: downgraded observation provenance to inference count=") {
		t.Fatalf("expected observation downgrade warning in run warnings, got %#v", info.Warnings)
	}
	if !hasWarningPrefix(info.Warnings, "refresh.step3.findings: semantic_guard: added fallback cross-repo edge") {
		t.Fatalf("expected fallback cross-repo edge warning in run warnings, got %#v", info.Warnings)
	}

	step1Taskruns, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-refresh-step1-collect-domain-*.json"))
	if err != nil {
		t.Fatalf("glob step1 taskruns: %v", err)
	}
	if len(step1Taskruns) == 0 {
		t.Fatalf("expected refresh step1 taskrun files")
	}
	step1Raw, err := os.ReadFile(step1Taskruns[len(step1Taskruns)-1])
	if err != nil {
		t.Fatalf("read step1 taskrun: %v", err)
	}
	var step1Payload contracts.TaskResult
	if err := json.Unmarshal(step1Raw, &step1Payload); err != nil {
		t.Fatalf("unmarshal step1 taskrun payload: %v", err)
	}
	step1Checked := false
	for _, op := range step1Payload.Changeset {
		if op.Op != "upsert_entity" || op.Entity == nil {
			continue
		}
		step1Checked = true
		if strings.TrimSpace(op.Entity.Provenance.Kind) != "inference" {
			t.Fatalf("expected downgraded inference provenance, got %q", op.Entity.Provenance.Kind)
		}
		for _, evidence := range op.Entity.Provenance.Evidence {
			if evidence.Path == "/" || evidence.Path == "." {
				t.Fatalf("expected invalid evidence paths removed, got %+v", op.Entity.Provenance.Evidence)
			}
		}
		break
	}
	if !step1Checked {
		t.Fatalf("expected at least one upsert_entity in step1 payload")
	}

	step3TaskrunCandidates, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-refresh-step3-findings*.json"))
	if err != nil {
		t.Fatalf("glob step3 taskruns: %v", err)
	}
	step3Taskruns := make([]string, 0, len(step3TaskrunCandidates))
	for _, candidate := range step3TaskrunCandidates {
		if strings.Contains(candidate, "shard-summary") {
			continue
		}
		step3Taskruns = append(step3Taskruns, candidate)
	}
	sort.Strings(step3Taskruns)
	if len(step3Taskruns) == 0 {
		t.Fatalf("expected refresh step3 taskrun file")
	}
	step3Raw, err := os.ReadFile(step3Taskruns[len(step3Taskruns)-1])
	if err != nil {
		t.Fatalf("read step3 taskrun: %v", err)
	}
	var step3Payload contracts.TaskResult
	if err := json.Unmarshal(step3Raw, &step3Payload); err != nil {
		t.Fatalf("unmarshal step3 taskrun payload: %v", err)
	}
	foundEdge := false
	for _, op := range step3Payload.Changeset {
		if op.Op != "upsert_edge" || op.Edge == nil {
			continue
		}
		foundEdge = true
		if strings.TrimSpace(op.Edge.From) == "" || strings.TrimSpace(op.Edge.To) == "" {
			t.Fatalf("expected edge endpoints to be non-empty, got %+v", op.Edge)
		}
		if strings.TrimSpace(op.Edge.From) == strings.TrimSpace(op.Edge.To) {
			t.Fatalf("expected cross-repo edge between different entities, got %+v", op.Edge)
		}
		if len(op.Edge.Provenance.Evidence) < 2 {
			t.Fatalf("expected fallback cross-repo edge to include evidence from two scopes, got %+v", op.Edge.Provenance.Evidence)
		}
		for _, evidence := range op.Edge.Provenance.Evidence {
			if evidence.Path == "/" || evidence.Path == "." {
				t.Fatalf("expected normalized edge evidence paths, got %+v", op.Edge.Provenance.Evidence)
			}
		}
		break
	}
	if !foundEdge {
		t.Fatalf("expected fallback upsert_edge in step3 payload, got %#v", step3Payload.Changeset)
	}
}

type delayedRunner struct {
	delay time.Duration
}

func (r delayedRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	select {
	case <-ctx.Done():
		return acpruntime.Result{}, ctx.Err()
	case <-time.After(r.delay):
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

type countingDelayedRunner struct {
	delay time.Duration

	mu    sync.Mutex
	calls map[string]int
}

func (r *countingDelayedRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	r.mu.Lock()
	if r.calls == nil {
		r.calls = map[string]int{}
	}
	r.calls[task.RunID]++
	r.mu.Unlock()

	select {
	case <-ctx.Done():
		return acpruntime.Result{}, ctx.Err()
	case <-time.After(r.delay):
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (r *countingDelayedRunner) callsForRun(runID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[runID]
}

type cancelReturnsRunnerUnavailableRunner struct{}

func (cancelReturnsRunnerUnavailableRunner) Run(ctx context.Context, _ acpruntime.Task) (acpruntime.Result, error) {
	<-ctx.Done()
	return acpruntime.Result{}, acpruntime.WrapRunnerError(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerUnavailable,
		"qwen-code runner is unavailable: signal: killed",
		errors.New("signal: killed"),
	)
}

func (cancelReturnsRunnerUnavailableRunner) Preflight(context.Context) error {
	return nil
}

type runtimeFailureWithOutputRunner struct{}

func (runtimeFailureWithOutputRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerUnavailable,
		"synthetic runtime failure",
		buildLongSnippet("stdout-synthetic"),
		buildLongSnippet("stderr-synthetic"),
		errors.New("synthetic runtime failure"),
	)
}

func (runtimeFailureWithOutputRunner) Preflight(context.Context) error {
	return nil
}

type runtimeParseFailureWithOutputRunner struct{}

func (runtimeParseFailureWithOutputRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	return acpruntime.Result{
		RawJSON: []byte(`{"meta":`),
		Stdout:  buildLongSnippet("stdout-parse"),
		Stderr:  buildLongSnippet("stderr-parse"),
	}, nil
}

func (runtimeParseFailureWithOutputRunner) Preflight(context.Context) error {
	return nil
}

func buildLongSnippet(prefix string) string {
	payload := prefix + "\r\n" + strings.Repeat(prefix+"-chunk-", 600)
	return payload
}

type step3ParseFailureRunner struct{}

func (step3ParseFailureRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if strings.HasSuffix(task.StepID, "step3.findings") {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			"synthetic parse failure at findings step",
			nil,
		)
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (step3ParseFailureRunner) Preflight(context.Context) error {
	return nil
}

type refreshCollectNoiseRunner struct{}

func (refreshCollectNoiseRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if task.StepID != "refresh.step1.collect" {
		return result, nil
	}

	taskResult := result.TaskResult
	taskResult.Changeset = append(taskResult.Changeset, contracts.Operation{
		Op: "upsert_entity",
		Entity: &contracts.Entity{
			ID:   "runtime.claude-code.orders-monolith",
			Type: "runtime_provider",
			Name: "Claude Runtime",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "orders-monolith", Path: "runtime-manifest.json"},
				},
			},
		},
	})

	raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
	if marshalErr != nil {
		return acpruntime.Result{}, marshalErr
	}
	result.TaskResult = taskResult
	result.RawJSON = raw
	return result, nil
}

func (refreshCollectNoiseRunner) Preflight(context.Context) error { return nil }

type refreshMissingFindingsRunner struct{}

func (refreshMissingFindingsRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if task.StepID != "refresh.step3.findings" {
		return result, nil
	}

	taskResult := result.TaskResult
	taskResult.Changeset = []contracts.Operation{}
	taskResult.Coverage = &contracts.Coverage{
		Observed: []string{"services"},
		Missing:  []string{"owner mappings", "ci-cd evidence", "delta validation"},
		Notes:    []string{"refresh evidence is incomplete"},
	}
	taskResult.Questions = []contracts.Question{
		{ID: "q.refresh.delta", Text: "What changed since previous run that affects ownership or dependencies?", Priority: "high"},
	}

	raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
	if marshalErr != nil {
		return acpruntime.Result{}, marshalErr
	}
	result.TaskResult = taskResult
	result.RawJSON = raw
	return result, nil
}

func (refreshMissingFindingsRunner) Preflight(context.Context) error { return nil }

type refreshNoServiceMissingFindingsRunner struct{}

func (refreshNoServiceMissingFindingsRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if task.StepID != "refresh.step1.collect" && task.StepID != "refresh.step3.findings" {
		return result, nil
	}

	taskResult := result.TaskResult
	taskResult.Changeset = []contracts.Operation{}
	taskResult.Coverage = &contracts.Coverage{
		Observed: []string{"services"},
		Missing:  []string{"owner mappings", "ci-cd evidence", "delta validation"},
		Notes:    []string{"refresh evidence is incomplete"},
	}
	taskResult.Questions = []contracts.Question{
		{ID: "q.refresh.delta", Text: "What changed since previous run that affects ownership or dependencies?", Priority: "high"},
	}

	raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
	if marshalErr != nil {
		return acpruntime.Result{}, marshalErr
	}
	result.TaskResult = taskResult
	result.RawJSON = raw
	return result, nil
}

func (refreshNoServiceMissingFindingsRunner) Preflight(context.Context) error { return nil }

type refreshCollectOffTopicRunner struct{}

func (refreshCollectOffTopicRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if task.StepID != "refresh.step1.collect" {
		return result, nil
	}

	taskResult := result.TaskResult
	taskResult.Changeset = append(taskResult.Changeset, contracts.Operation{
		Op: "upsert_entity",
		Entity: &contracts.Entity{
			ID:   "external.chinabidding",
			Type: "external.system",
			Name: "China Bidding Network",
			Attributes: map[string]any{
				"url": "https://www.chinabidding.cn",
			},
			Provenance: contracts.Provenance{
				Kind:       "inference",
				Confidence: 0.4,
				Evidence: []contracts.Evidence{
					{Repo: "orders-monolith", Path: "search_source/chinabidding.cn"},
				},
			},
		},
	})
	taskResult.Questions = append(taskResult.Questions, contracts.Question{
		ID:       "q.refresh.delta.1",
		Text:     "What bidding announcements were published since last run?",
		Priority: "high",
	})

	raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
	if marshalErr != nil {
		return acpruntime.Result{}, marshalErr
	}
	result.TaskResult = taskResult
	result.RawJSON = raw
	return result, nil
}

func (refreshCollectOffTopicRunner) Preflight(context.Context) error { return nil }

type refreshCollectCriticalOffTopicRunner struct{}

func (refreshCollectCriticalOffTopicRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if task.StepID != "refresh.step1.collect" {
		return result, nil
	}

	taskResult := result.TaskResult
	taskResult.Changeset = []contracts.Operation{
		{
			Op: "upsert_entity",
			Entity: &contracts.Entity{
				ID:   "external.chinabidding",
				Type: "external.system",
				Name: "China Bidding Network",
				Attributes: map[string]any{
					"url": "https://www.chinabidding.cn",
				},
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.4,
					Evidence: []contracts.Evidence{
						{Repo: "orders-monolith", Path: "search_source/chinabidding.cn"},
					},
				},
			},
		},
	}
	taskResult.Questions = []contracts.Question{
		{
			ID:       "q.refresh.delta.critical",
			Text:     "What bidding announcements were published since last run?",
			Priority: "high",
		},
	}

	raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
	if marshalErr != nil {
		return acpruntime.Result{}, marshalErr
	}
	result.TaskResult = taskResult
	result.RawJSON = raw
	return result, nil
}

func (refreshCollectCriticalOffTopicRunner) Preflight(context.Context) error { return nil }

type refreshInvalidEvidenceRunner struct{}

func (refreshInvalidEvidenceRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if task.StepID != "refresh.step1.collect" {
		return result, nil
	}

	taskResult := result.TaskResult
	for idx := range taskResult.Changeset {
		if taskResult.Changeset[idx].Op != "upsert_entity" || taskResult.Changeset[idx].Entity == nil {
			continue
		}
		taskResult.Changeset[idx].Entity.Provenance = contracts.Provenance{
			Kind:       "observation",
			Confidence: 0.82,
			Evidence: []contracts.Evidence{
				{Repo: "orders-monolith", Path: "/"},
				{Repo: "orders-monolith", Path: "."},
			},
		}
		break
	}

	raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
	if marshalErr != nil {
		return acpruntime.Result{}, marshalErr
	}
	result.TaskResult = taskResult
	result.RawJSON = raw
	return result, nil
}

func (refreshInvalidEvidenceRunner) Preflight(context.Context) error { return nil }

type refreshMultiScopeNoEdgeInvalidEvidenceRunner struct{}

func (refreshMultiScopeNoEdgeInvalidEvidenceRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}

	taskResult := result.TaskResult
	switch task.StepID {
	case "refresh.step1.collect":
		for idx := range taskResult.Changeset {
			if taskResult.Changeset[idx].Op != "upsert_entity" || taskResult.Changeset[idx].Entity == nil {
				continue
			}
			taskResult.Changeset[idx].Entity.Provenance = contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.82,
				Evidence: []contracts.Evidence{
					{Repo: "payments-service", Path: "/"},
					{Repo: "users-service", Path: "."},
				},
			}
			break
		}
	case "refresh.step3.findings":
		withoutEdges := make([]contracts.Operation, 0, len(taskResult.Changeset))
		for _, op := range taskResult.Changeset {
			if op.Op == "upsert_edge" {
				continue
			}
			withoutEdges = append(withoutEdges, op)
		}
		taskResult.Changeset = withoutEdges
	}

	raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
	if marshalErr != nil {
		return acpruntime.Result{}, marshalErr
	}
	result.TaskResult = taskResult
	result.RawJSON = raw
	return result, nil
}

func (refreshMultiScopeNoEdgeInvalidEvidenceRunner) Preflight(context.Context) error { return nil }

type docArtifactRunner struct{}

func (docArtifactRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step1.collect") {
		taskResult := result.TaskResult
		taskResult.Changeset = append(taskResult.Changeset, contracts.Operation{
			Op: "add_doc_artifact",
			DocArtifact: &contracts.DocArtifact{
				ID:     "imports.architecture.notes",
				Kind:   "imported-doc",
				Title:  "Architecture Notes",
				Path:   "docs/imports/architecture-notes.md",
				Format: "markdown",
			},
		})
		raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
		if marshalErr != nil {
			return acpruntime.Result{}, marshalErr
		}
		result.TaskResult = taskResult
		result.RawJSON = raw
	}
	return result, nil
}

func createWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repoA := filepath.Join(root, "repos", "payments-service")
	repoB := filepath.Join(root, "repos", "users-service")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("create payments repo: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("create users repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoA, "README.md"), []byte("# payments-service\n"), 0o644); err != nil {
		t.Fatalf("write payments readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoB, "README.md"), []byte("# users-service\n"), 0o644); err != nil {
		t.Fatalf("write users readme: %v", err)
	}
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoA + `
  - name: users-service
    path: ` + repoB + `
`
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func createWorkspaceWithTimeouts(t *testing.T, timeouts map[string]int) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repoA := filepath.Join(root, "repos", "payments-service")
	repoB := filepath.Join(root, "repos", "users-service")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("create payments repo: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("create users repo: %v", err)
	}
	manifest := strings.Builder{}
	manifest.WriteString("version: 1\nrepos:\n")
	manifest.WriteString("  - name: payments-service\n")
	manifest.WriteString("    path: " + repoA + "\n")
	manifest.WriteString("  - name: users-service\n")
	manifest.WriteString("    path: " + repoB + "\n")
	manifest.WriteString("runtime:\n")
	manifest.WriteString("  profile:\n")
	manifest.WriteString("    timeouts:\n")
	for _, key := range []string{
		"step_timeout_sec",
		"heartbeat_sec",
		"pipeline_timeout_sec",
		"pipeline_kill_grace_sec",
		"api_ready_timeout_sec",
		"api_init_timeout_sec",
		"ui_init_poll_timeout_sec",
		"ui_cancel_poll_timeout_sec",
	} {
		if value, ok := timeouts[key]; ok && value > 0 {
			manifest.WriteString(fmt.Sprintf("      %s: %d\n", key, value))
		}
	}

	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest.String()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func createMonolithWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repo := filepath.Join(root, "repos", "orders-monolith")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("create monolith repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# orders-monolith\n"), 0o644); err != nil {
		t.Fatalf("write monolith readme: %v", err)
	}
	manifest := `version: 1
repos:
  - name: orders-monolith
    path: ` + repo + `
`
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func waitForRunTerminalState(t *testing.T, service *Service, runID string, timeout time.Duration) RunInfo {
	t.Helper()

	var terminal RunInfo
	testutil.WaitFor(t, timeout, testutil.WaitDescription("run %q did not reach terminal status", runID), func() (bool, error) {
		info, ok := service.GetRun(runID)
		if ok && (info.Status == RunStatusSucceeded || info.Status == RunStatusFailed) {
			terminal = info
			return true, nil
		}
		return false, nil
	})
	return terminal
}

func waitForAsyncDrain(t *testing.T, service *Service, timeout time.Duration) {
	t.Helper()

	testutil.WaitFor(t, timeout, testutil.WaitDescription("async runs did not drain"), func() (bool, error) {
		service.mu.RLock()
		defer service.mu.RUnlock()

		if strings.TrimSpace(service.activeRunID) != "" || service.pendingRun != nil {
			return false, nil
		}
		for _, record := range service.runs {
			if record == nil {
				continue
			}
			if record.info.Status == RunStatusQueued || record.info.Status == RunStatusRunning {
				return false, nil
			}
		}
		return true, nil
	})
}

func runRegistrySize(service *Service) int {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return len(service.runs)
}

func assertRunRegistryContainsOnly(t *testing.T, service *Service, runIDs ...string) {
	t.Helper()
	expected := map[string]struct{}{}
	for _, runID := range runIDs {
		expected[runID] = struct{}{}
	}

	service.mu.RLock()
	defer service.mu.RUnlock()

	if len(service.runs) != len(expected) {
		t.Fatalf("unexpected run registry size: got %d want %d", len(service.runs), len(expected))
	}
	for runID := range service.runs {
		if _, ok := expected[runID]; !ok {
			t.Fatalf("unexpected run registry entry %q", runID)
		}
	}
}

func waitForRunHistoryStatus(t *testing.T, ws workspace.Root, runID string, timeout time.Duration, wanted RunStatus) RunStatus {
	t.Helper()

	var observed RunStatus
	testutil.WaitFor(t, timeout, testutil.WaitDescription("run %q did not reach history status %q", runID, wanted), func() (bool, error) {
		snapshot, err := loadRunHistorySnapshot(ws)
		if err != nil {
			return false, nil
		}
		for _, item := range snapshot.Items {
			if item.RunID != runID {
				continue
			}
			observed = item.Status
			return item.Status == wanted, nil
		}
		return false, nil
	})
	return observed
}

func loadRunHistorySnapshot(ws workspace.Root) (runHistorySnapshot, error) {
	content, err := os.ReadFile(filepath.Join(ws.Path, runHistoryPath))
	if err != nil {
		return runHistorySnapshot{}, err
	}
	var snapshot runHistorySnapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		return runHistorySnapshot{}, err
	}
	return snapshot, nil
}

func writeShardRecoveryMarker(t *testing.T, ws workspace.Root, runID string, stepID string) {
	t.Helper()

	taskrunsDir := filepath.Join(ws.Path, "reports", "taskruns")
	if err := os.MkdirAll(taskrunsDir, 0o755); err != nil {
		t.Fatalf("create taskruns dir: %v", err)
	}
	summary := runtimeShardSummary{
		Version:       1,
		RunID:         runID,
		StepID:        stepID,
		Strategy:      "parallel",
		MaxParallel:   1,
		FailurePolicy: "best_effort",
		ShardMode:     "heuristics",
		GeneratedAt:   time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Items: []runtimeShardSummaryEntry{
			{
				ShardID: "synthetic-shard",
				Status:  "pending",
			},
		},
	}
	content, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("marshal shard recovery marker: %v", err)
	}
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	path := filepath.Join(taskrunsDir, fmt.Sprintf("%s-%s-shard-summary.json", runID, stepSlug))
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		t.Fatalf("write shard recovery marker: %v", err)
	}
}

func hasWarningPrefix(warnings []string, prefix string) bool {
	for _, warning := range warnings {
		if strings.HasPrefix(strings.TrimSpace(warning), prefix) {
			return true
		}
	}
	return false
}

func findRunLogByMessage(t *testing.T, entries []RunLogEntry, message string) RunLogEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.Message == message {
			return entry
		}
	}
	t.Fatalf("expected run log message %q, got %+v", message, entries)
	return RunLogEntry{}
}

type trackingRunner struct {
	mu    sync.Mutex
	tasks []acpruntime.Task
}

func (r *trackingRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	r.mu.Lock()
	r.tasks = append(r.tasks, task)
	r.mu.Unlock()
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (r *trackingRunner) tasksForStep(stepID string) []acpruntime.Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	filtered := []acpruntime.Task{}
	for _, task := range r.tasks {
		if task.StepID == stepID {
			filtered = append(filtered, task)
		}
	}
	return filtered
}
