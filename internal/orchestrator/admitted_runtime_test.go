package orchestrator

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
)

func TestRunUsesAdmittedRuntimeSnapshotForEveryProviderTask(t *testing.T) {
	t.Parallel()
	ws := createWorkspace(t)
	var mu sync.Mutex
	var observed []acpruntime.Task
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunnerFactory(func(provider acpruntime.Provider) (acpruntime.Runner, error) {
			return recordingSnapshotRunner{provider: provider, mu: &mu, observed: &observed}, nil
		}),
	)
	steps := map[string]acpruntime.Provider{}
	for _, step := range []string{
		acpruntime.StepProviderStep0Constitution,
		acpruntime.StepProviderStep1Collect,
		acpruntime.StepProviderStep2AsIs,
		acpruntime.StepProviderStep3Findings,
		acpruntime.StepProviderStep4Proposals,
	} {
		steps[step] = acpruntime.ProviderCodexCode
	}
	snapshot := &acpruntime.AdmittedRuntimeSnapshot{
		Mode:                 acpruntime.RuntimeModeFake,
		StepProviders:        steps,
		StepProviderSources:  acpruntime.StepProviderSources{acpruntime.StepProviderStep1Collect: acpruntime.ProviderSourceOverride},
		ProviderModels:       acpruntime.ProviderModelValues{acpruntime.ProviderCodexCode: {Model: "luna", Effort: "high"}},
		ProviderModelSources: acpruntime.ProviderModelSources{acpruntime.ProviderCodexCode: {Model: acpruntime.ProviderModelSourceEnv, Effort: acpruntime.ProviderModelSourceEnv}},
		Execution:            acpruntime.ExecutionValues{Strategy: acpruntime.ExecutionStrategyParallel, MaxParallel: 7, FailurePolicy: acpruntime.ExecutionFailurePolicyFailFast, ShardMode: acpruntime.ExecutionShardDiscoverySemantic},
		Permissions:          acpruntime.PermissionValues{Mode: acpruntime.PermissionModeManaged, ApprovalChannel: acpruntime.PermissionApprovalUI},
		Timeouts:             acpruntime.TimeoutValues{StepTimeoutSec: 41, HeartbeatSec: 9, PipelineTimeoutSec: 71, PipelineKillGraceSec: 11, APIReadyTimeoutSec: 13, APIInitTimeoutSec: 17, UIInitPollTimeoutSec: 19, UICancelPollTimeoutSec: 23},
	}
	info, _, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineInit, RuntimeSnapshot: snapshot, NonInteractive: true})
	if err != nil {
		t.Fatalf("run with admitted snapshot: %v", err)
	}
	if info.StepProviders[acpruntime.StepProviderStep1Collect] != string(acpruntime.ProviderCodexCode) || info.ExecutionProfile != snapshot.Execution || info.PermissionProfile != snapshot.Permissions || info.Timeouts != snapshot.Timeouts {
		t.Fatalf("run info lost admitted snapshot: %+v", info)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(observed) == 0 {
		t.Fatal("expected provider tasks")
	}
	for _, task := range observed {
		if task.RuntimeModel != "luna" || task.RuntimeEffort != "high" || task.RuntimePermissions != snapshot.Permissions {
			t.Fatalf("provider task used mutable/default runtime settings: %+v", task)
		}
	}
}

func TestQueuedRunCopiesAdmittedRuntimeSnapshotBeforeCallerMutation(t *testing.T) {
	t.Parallel()
	ws := createWorkspace(t)
	release := make(chan struct{})
	runner := &countingBlockingRunner{release: release}
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(runner),
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

	steps := map[string]acpruntime.Provider{}
	for _, step := range []string{
		acpruntime.StepProviderStep0Constitution,
		acpruntime.StepProviderStep1Collect,
		acpruntime.StepProviderStep2AsIs,
		acpruntime.StepProviderStep3Findings,
		acpruntime.StepProviderStep4Proposals,
	} {
		steps[step] = acpruntime.ProviderClaudeCode
	}
	snapshot := &acpruntime.AdmittedRuntimeSnapshot{
		Mode:                 acpruntime.RuntimeModeFake,
		StepProviders:        steps,
		RepositoryScopes:     []string{"original-scope"},
		RepositoryPathScopes: map[string][]string{"original-scope": {"src", "docs"}},
	}
	queuedRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:       ws,
		Pipeline:        PipelineRefresh,
		Intent:          RunIntentQueue,
		RuntimeSnapshot: snapshot,
		NonInteractive:  true,
	})
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	if queuedRunID == activeRunID {
		t.Fatalf("expected distinct queued run id")
	}
	snapshot.RepositoryScopes[0] = "mutated-after-admission"
	snapshot.RepositoryPathScopes["original-scope"][0] = "mutated-after-admission"

	close(release)
	queued := waitForRunTerminalInfo(t, service, queuedRunID, 5*time.Second)
	if len(queued.RepositoryScopes) != 1 || queued.RepositoryScopes[0] != "original-scope" {
		t.Fatalf("queued run adopted caller mutation: %+v", queued.RepositoryScopes)
	}
	if got := queued.RepositoryPathScopes["original-scope"]; !reflect.DeepEqual(got, []string{"src", "docs"}) {
		t.Fatalf("queued run adopted caller path-scope mutation: %+v", got)
	}
	waitForServiceQuiescent(t, service, 2*time.Second)
}

func TestRunHistoryRoundTripsRepositoryPathScopes(t *testing.T) {
	t.Parallel()
	started := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	record := runRecord{info: RunInfo{
		RunID:                "run-path-scope",
		Pipeline:             string(PipelineInit),
		Status:               RunStatusRunning,
		StartedAt:            started,
		RepositoryScopes:     []string{"payments-service"},
		RepositoryPathScopes: map[string][]string{"payments-service": {"src", "docs/**/*.md"}},
	}}
	item := runRecordToHistoryItem(record)
	restored, ok := historyItemToRunRecord(item)
	if !ok {
		t.Fatal("history item did not restore")
	}
	if !reflect.DeepEqual(restored.info.RepositoryPathScopes, record.info.RepositoryPathScopes) {
		t.Fatalf("repository path scopes changed across history round-trip: got=%v want=%v", restored.info.RepositoryPathScopes, record.info.RepositoryPathScopes)
	}
}

type recordingSnapshotRunner struct {
	provider acpruntime.Provider
	mu       *sync.Mutex
	observed *[]acpruntime.Task
}

func (r recordingSnapshotRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	r.mu.Lock()
	*r.observed = append(*r.observed, task)
	r.mu.Unlock()
	return fakeruntime.Runner{}.Run(ctx, task)
}
