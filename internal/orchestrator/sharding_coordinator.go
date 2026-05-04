package orchestrator

import (
	"context"
	"fmt"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func (e *pipelineExecution) executeRuntimeTasksSharded(
	ctx context.Context,
	stepID string,
	domainID string,
	repoScopes []string,
	taskSuffixPrefix string,
) ([]runtimeTaskExecution, runtimeShardOutcome, error) {
	plans, plannerWarnings, semanticGraph := e.planRuntimeShards(repoScopes)
	for _, warning := range plannerWarnings {
		message := strings.TrimSpace(warning)
		if message == "" {
			continue
		}
		e.addWarning(fmt.Sprintf("%s: %s", stepID, message))
		e.logWarn(stepID, domainID, "runtime shard planner warning", map[string]any{"warning": message})
	}
	if len(plans) == 0 {
		plans = []runtimeShardPlan{{
			SortKey:     "workspace:.",
			ShardID:     "workspace-root",
			RepoScopes:  append([]string(nil), repoScopes...),
			PathScopes:  []string{"."},
			PrimaryRepo: "workspace",
		}}
	}

	options := normalizeRuntimeShardExecutionOptions(e.executionProfile)
	if err := e.persistShardPlan(stepID, domainID, plans, plannerWarnings, semanticGraph, options.Strategy, options.MaxParallel, options.FailurePolicy); err != nil {
		return nil, runtimeShardOutcome{}, err
	}

	singleShard := len(plans) == 1
	summaryState, err := e.loadRuntimeShardSummaryState(stepID, domainID, plans, singleShard)
	if err != nil {
		return nil, runtimeShardOutcome{}, err
	}

	e.logInfo(stepID, domainID, "runtime shard execution prepared", map[string]any{
		"shards":         len(plans),
		"strategy":       options.Strategy,
		"max_parallel":   options.MaxParallel,
		"failure_policy": options.FailurePolicy,
		"shard_mode":     e.executionProfile.ShardMode,
	})

	results, terminalErr := e.scheduleRuntimeShardRuns(ctx, stepID, domainID, plans, summaryState, options, taskSuffixPrefix)
	if terminalErr != nil && !options.BestEffort {
		e.logWarn(stepID, domainID, "runtime shard scheduler stopped after terminal failure", map[string]any{
			"error": strings.TrimSpace(terminalErr.Error()),
		})
	}

	executions := make([]runtimeTaskExecution, 0, len(plans))
	outcome := runtimeShardOutcome{PlannedShards: len(plans)}

	for _, result := range results {
		if result.Err == nil && result.Prepared.Task.TaskID == "" && terminalErr != nil && !options.BestEffort {
			if err := summaryState.markAborted(result.Plan); err != nil {
				return nil, runtimeShardOutcome{}, err
			}
			outcome.FailedShards++
			continue
		}
		if result.Err != nil {
			outcome.FailedShards++
			if options.BestEffort {
				e.registerPartialShardFailure(stepID, domainID, result.Plan, result.Err)
				continue
			}
			if terminalErr == nil {
				terminalErr = result.Err
			}
			continue
		}

		var execution runtimeTaskExecution
		if result.AlreadyApplied && !strings.HasSuffix(stepID, "step3.findings") {
			execution, err = e.replayRuntimeTaskExecution(stepID, domainID, result.Prepared)
		} else {
			execution, err = e.applyRuntimeTaskExecution(stepID, domainID, result.Prepared)
		}
		if err != nil {
			outcome.FailedShards++
			if markErr := summaryState.markFailed(result.Plan, result.Prepared.Task.TaskID, strings.TrimSpace(err.Error())); markErr != nil {
				return nil, runtimeShardOutcome{}, markErr
			}
			if options.BestEffort {
				e.registerPartialShardFailure(stepID, domainID, result.Plan, err)
				continue
			}
			if terminalErr == nil {
				terminalErr = err
			}
			continue
		}

		taskrunPath := shardTaskrunPath(e.runID, stepID, domainID, result.Plan.ShardID, summaryState.singleShard)
		if err := summaryState.markSucceeded(result.Plan, result.Prepared.Task.TaskID, taskrunPath); err != nil {
			return nil, runtimeShardOutcome{}, err
		}
		if result.FromCheckpoint {
			e.logInfo(stepID, domainID, "shard replayed from checkpoint", map[string]any{
				"shard_id":     result.Plan.ShardID,
				"task_id":      result.Prepared.Task.TaskID,
				"taskrun_path": taskrunPath,
			})
		}
		outcome.SucceededShards++
		executions = append(executions, execution)
	}

	if terminalErr != nil {
		return nil, outcome, terminalErr
	}
	return executions, outcome, nil
}

func (e *pipelineExecution) registerPartialShardFailure(stepID string, domainID string, plan runtimeShardPlan, err error) {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "unknown shard error"
	}
	e.partialFailures = append(e.partialFailures, runtimeShardFailure{
		StepID:     stepID,
		DomainID:   domainID,
		ShardID:    plan.ShardID,
		RepoScopes: append([]string(nil), plan.RepoScopes...),
		PathScopes: append([]string(nil), plan.PathScopes...),
		Message:    message,
	})
	e.addWarning(fmt.Sprintf("%s: shard %q failed (%s)", stepID, plan.ShardID, message))
	e.logError(stepID, domainID, "runtime shard failed (best-effort continues)", map[string]any{
		"shard_id":    plan.ShardID,
		"repo_scopes": plan.RepoScopes,
		"path_scopes": plan.PathScopes,
		"error":       message,
	})
}

func normalizeRuntimeShardExecutionOptions(profile acpruntime.ExecutionValues) runtimeShardExecutionOptions {
	strategy := strings.TrimSpace(profile.Strategy)
	if strategy == "" {
		strategy = "sequential"
	}
	maxParallel := profile.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 1
	}
	if strategy != "parallel" {
		maxParallel = 1
	}
	failurePolicy := strings.TrimSpace(profile.FailurePolicy)
	if failurePolicy == "" {
		failurePolicy = "best_effort"
	}
	return runtimeShardExecutionOptions{
		Strategy:      strategy,
		MaxParallel:   maxParallel,
		FailurePolicy: failurePolicy,
		BestEffort:    failurePolicy == "best_effort",
	}
}
