package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const collectBestEffortRunnerUnavailableAbortThreshold = 5

func (e *pipelineExecution) scheduleRuntimeShardRuns(
	ctx context.Context,
	stepID string,
	domainID string,
	plans []runtimeShardPlan,
	summaryState *runtimeShardSummaryState,
	options runtimeShardExecutionOptions,
	taskSuffixPrefix string,
) ([]runtimeShardRunResult, error) {
	return defaultShardScheduler{execution: e}.ScheduleRuntimeShardRuns(ctx, ShardScheduleRequest{
		StepID:           stepID,
		DomainID:         domainID,
		Plans:            plans,
		SummaryState:     summaryState,
		Options:          options,
		TaskSuffixPrefix: taskSuffixPrefix,
	})
}

func (s defaultShardScheduler) ScheduleRuntimeShardRuns(ctx context.Context, request ShardScheduleRequest) ([]runtimeShardRunResult, error) {
	plans := request.Plans
	options := request.Options
	results := make([]runtimeShardRunResult, len(plans))
	for idx, plan := range plans {
		results[idx].Plan = plan
	}
	if len(plans) == 0 {
		return results, nil
	}

	workerCount := options.MaxParallel
	if workerCount <= 0 {
		workerCount = 1
	}
	if workerCount > len(plans) {
		workerCount = len(plans)
	}

	runCtx := ctx
	cancel := func() {}
	abortOnCollectRunnerUnavailable := options.BestEffort &&
		workerCount == 1 &&
		acpruntime.IsCollectStep(request.StepID)
	if !options.BestEffort || abortOnCollectRunnerUnavailable {
		runCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var terminalErr error
	consecutiveRunnerUnavailable := 0
	nextIndex := 0

	recordResult := func(index int, result runtimeShardRunResult) {
		mu.Lock()
		defer mu.Unlock()
		results[index] = result
		if result.Err == nil {
			consecutiveRunnerUnavailable = 0
			return
		}
		if !options.BestEffort && terminalErr == nil {
			terminalErr = result.Err
			cancel()
			return
		}
		if abortOnCollectRunnerUnavailable {
			if shardErrorCode(result.Err) == string(acpruntime.ErrorCodeRunnerUnavailable) {
				consecutiveRunnerUnavailable++
			} else {
				consecutiveRunnerUnavailable = 0
			}
			if consecutiveRunnerUnavailable >= collectBestEffortRunnerUnavailableAbortThreshold && terminalErr == nil {
				terminalErr = fmt.Errorf("collect aborted after %d consecutive runner_unavailable shards: %w", consecutiveRunnerUnavailable, result.Err)
				cancel()
			}
		}
	}
	nextJob := func() (int, runtimeShardPlan, bool) {
		mu.Lock()
		defer mu.Unlock()
		if terminalErr != nil {
			return 0, runtimeShardPlan{}, false
		}
		if nextIndex >= len(plans) {
			return 0, runtimeShardPlan{}, false
		}
		index := nextIndex
		nextIndex++
		return index, plans[index], true
	}

	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				index, plan, ok := nextJob()
				if !ok {
					return
				}
				result := s.execution.runRuntimeShard(runCtx, request.StepID, request.DomainID, plan, request.SummaryState, request.TaskSuffixPrefix)
				recordResult(index, result)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	return results, terminalErr
}

func (e *pipelineExecution) runRuntimeShard(
	ctx context.Context,
	stepID string,
	domainID string,
	plan runtimeShardPlan,
	summaryState *runtimeShardSummaryState,
	taskSuffixPrefix string,
) runtimeShardRunResult {
	entry := summaryState.entry(plan.ShardID)
	if handled, result := e.loadReplayableShardResult(stepID, domainID, plan, entry, summaryState.singleShard); handled {
		if result.Err != nil {
			taskID := strings.TrimSpace(entry.TaskID)
			if markErr := summaryState.markFailedError(plan, taskID, result.Err); markErr != nil {
				result.Err = markErr
			}
		}
		return result
	}
	if entry.Status == "failed" {
		return runtimeShardRunResult{
			Plan: plan,
			Err:  shardFailureError(entry),
		}
	}

	taskSuffix := buildShardTaskSuffix(taskSuffixPrefix, plan.ShardID)
	prepared, err := e.runRuntimeTaskNormalized(ctx, stepID, taskSuffix, plan.RepoScopes, plan.PathScopes, domainID, plan.ShardID)
	if err == nil {
		taskrunPath := shardTaskrunPath(e.runID, stepID, domainID, plan.ShardID, summaryState.singleShard)
		taskrunLabel := shardTaskrunLabel(stepID, domainID, plan.ShardID, summaryState.singleShard)
		if checkpointErr := summaryState.markCheckpointed(plan, prepared.Task.TaskID, taskrunPath, taskrunLabel, prepared.ExecutionRaw); checkpointErr != nil {
			err = checkpointErr
		}
	}
	if err != nil {
		if markErr := summaryState.markFailedError(plan, prepared.Task.TaskID, err); markErr != nil {
			err = markErr
		}
	}
	return runtimeShardRunResult{Plan: plan, Prepared: prepared, Err: err}
}
