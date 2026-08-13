package orchestrator

import (
	"context"
	"sync"
	"testing"

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
