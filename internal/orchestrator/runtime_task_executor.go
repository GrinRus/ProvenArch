package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

type RuntimeTaskRequest struct {
	StepID     string
	TaskSuffix string
	RepoScopes []string
	PathScopes []string
	DomainID   string
	ShardID    string
}

type RuntimeTaskExecutor interface {
	RunRuntimeTask(ctx context.Context, request RuntimeTaskRequest) (runtimePreparedExecution, error)
}

type defaultRuntimeTaskExecutor struct {
	execution *pipelineExecution
}

func (executor defaultRuntimeTaskExecutor) RunRuntimeTask(ctx context.Context, request RuntimeTaskRequest) (runtimePreparedExecution, error) {
	e := executor.execution
	if e == nil {
		return runtimePreparedExecution{}, fmt.Errorf("runtime task executor is not configured")
	}
	stepID := strings.TrimSpace(request.StepID)
	domainID := strings.TrimSpace(request.DomainID)
	shardID := strings.TrimSpace(request.ShardID)
	taskSuffix := strings.TrimSpace(request.TaskSuffix)
	repoScopes := append([]string(nil), request.RepoScopes...)
	pathScopes := append([]string(nil), request.PathScopes...)

	taskID := fmt.Sprintf("task-%s-%s", e.runID, strings.ReplaceAll(stepID, ".", "-"))
	if taskSuffix != "" {
		taskID += "-" + taskSuffix
	}
	repoScope := primaryRepoScope(repoScopes)
	artifactRootRel, writeRootAbs, draftFinalRootAbs, readContextRoots, err := e.runtimeArtifactContext(stepID, shardID, repoScopes)
	if err != nil {
		return runtimePreparedExecution{}, err
	}
	resolvedProvider := acpruntime.ProviderClaudeCode
	runner := acpruntime.Runner(nil)
	if e.runnerResolver != nil {
		resolvedProvider, runner, err = e.runnerResolver.ReadyRunnerForStep(ctx, stepID)
		if err != nil {
			return runtimePreparedExecution{}, err
		}
	}
	task := acpruntime.Task{
		TaskID:            taskID,
		RunID:             e.runID,
		StepID:            stepID,
		ShardID:           shardID,
		DomainID:          domainID,
		Workspace:         e.workspace.Path,
		ArtifactRoot:      artifactRootRel,
		WriteRoot:         writeRootAbs,
		DraftFinalRoot:    draftFinalRootAbs,
		ReadContextRoots:  append([]string(nil), readContextRoots...),
		AgentRole:         runtimeAgentRole(stepID),
		StepContract:      runtimeStepContract(stepID),
		ExpectedArtifacts: append([]string(nil), runtimeExpectedArtifacts(stepID)...),
		RepoScope:         repoScope,
		RepoScopes:        repoScopes,
		PathScopes:        pathScopes,
		StartedAtUTC:      e.clock().UTC(),
		RuntimeTimeoutProfile: map[string]any{
			"step_timeout_sec":      int(e.runtimeStepTimeout.Seconds()),
			"heartbeat_timeout_sec": int(e.runtimeHeartbeatInterval.Seconds()),
		},
		OnOutput: func(chunk acpruntime.OutputChunk) {
			e.logRuntimeOutput(stepID, domainID, resolvedProvider, chunk)
		},
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			e.recordRuntimeDiagnosticCounters(event)
			e.logInfo(stepID, domainID, event.Message, event.Fields)
		},
	}
	e.logInfo(stepID, domainID, "runtime task started", map[string]any{
		"task_id":            task.TaskID,
		"shard_id":           task.ShardID,
		"repo_scope":         task.RepoScope,
		"repo_scopes":        task.RepoScopes,
		"path_scopes":        task.PathScopes,
		"provider":           resolvedProvider,
		"artifact_root":      task.ArtifactRoot,
		"write_root":         task.WriteRoot,
		"draft_final_root":   task.DraftFinalRoot,
		"read_context_roots": task.ReadContextRoots,
	})

	taskCtx := ctx
	cancel := func() {}
	if e.runtimeStepTimeout > 0 {
		taskCtx, cancel = context.WithTimeout(ctx, e.runtimeStepTimeout)
	}
	defer cancel()

	if runner == nil {
		return runtimePreparedExecution{}, fmt.Errorf("runtime runner resolver is not configured")
	}

	var heartbeatStop chan struct{}
	var heartbeatWG sync.WaitGroup
	if e.runtimeHeartbeatInterval > 0 {
		heartbeatStop = make(chan struct{})
		heartbeatTicker := time.NewTicker(e.runtimeHeartbeatInterval)
		startedAt := e.clock().UTC()
		heartbeatWG.Add(1)
		go func() {
			defer heartbeatWG.Done()
			defer heartbeatTicker.Stop()
			for {
				select {
				case <-heartbeatStop:
					return
				case <-heartbeatTicker.C:
					e.logInfo(stepID, domainID, "runtime task heartbeat", map[string]any{
						"task_id":     task.TaskID,
						"shard_id":    task.ShardID,
						"repo_scope":  task.RepoScope,
						"repo_scopes": task.RepoScopes,
						"path_scopes": task.PathScopes,
						"elapsed_sec": int(time.Since(startedAt).Seconds()),
					})
				}
			}
		}()
	}
	stopHeartbeat := func() {
		if heartbeatStop != nil {
			close(heartbeatStop)
		}
		heartbeatWG.Wait()
		heartbeatStop = nil
	}
	defer stopHeartbeat()

	writeAudit := beginRuntimeWriteAudit(task)
	result, err := runner.Run(taskCtx, task)
	e.completeRuntimeWriteAudit(stepID, domainID, task, writeAudit)
	if err != nil {
		if isDraftOnlyRuntimeStep(stepID) {
			if _, _, draftErr := validateRequiredRuntimeDraftArtifacts(task); draftErr != nil {
				e.logError(stepID, domainID, "runtime draft artifact validation failed", map[string]any{
					"task_id": task.TaskID,
					"error":   strings.TrimSpace(draftErr.Error()),
				})
				err = fmt.Errorf("%w: required runtime draft artifacts invalid: %v", err, draftErr)
			}
		}
		if errors.Is(taskCtx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("runtime task timeout after %ds: %w", int(e.runtimeStepTimeout.Seconds()), err)
		}
		if failedExecution, ok := runtimeExecutionFromFailure(task, resolvedProvider, err, e.clock().UTC()); ok {
			failedExecutionLabel := stepID + ".runtime-execution"
			if raw, marshalErr := json.MarshalIndent(failedExecution, "", "  "); marshalErr != nil {
				e.logWarn(stepID, domainID, "marshal failed runtime execution metadata failed", map[string]any{
					"task_id": task.TaskID,
					"error":   marshalErr.Error(),
				})
			} else if persistErr := e.persistRuntimeExecutionArtifact(runtimeExecutionMetadataPathForTask(task), failedExecutionLabel, raw); persistErr != nil {
				e.logWarn(stepID, domainID, "persist failed runtime execution metadata failed", map[string]any{
					"task_id": task.TaskID,
					"error":   persistErr.Error(),
				})
			}
		}
		e.logError(stepID, domainID, "runtime task failed", runtimeFailureLogFields(task, err, "", ""))
		return runtimePreparedExecution{}, err
	}
	execution := contracts.NormalizeRuntimeExecution(result.Execution)
	if strings.TrimSpace(execution.TaskID) == "" {
		execution = acpruntime.NewExecution(task, resolvedProvider, "", "succeeded", e.clock().UTC(), nil)
	}
	if isDraftOnlyRuntimeStep(stepID) {
		if _, _, draftErr := validateRequiredRuntimeDraftArtifacts(task); draftErr != nil {
			e.logError(stepID, domainID, "runtime draft artifact validation failed", map[string]any{
				"task_id": task.TaskID,
				"error":   strings.TrimSpace(draftErr.Error()),
			})
			return runtimePreparedExecution{}, fmt.Errorf("runtime required draft artifacts invalid: %w", draftErr)
		}
	}
	executionRaw, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return runtimePreparedExecution{}, fmt.Errorf("marshal runtime execution: %w", err)
	}
	runtimeName := strings.TrimSpace(execution.Provider)
	runtimeVersion := strings.TrimSpace(execution.RuntimeVersion)
	if runtimeName == "" {
		runtimeName = string(resolvedProvider)
	}

	return runtimePreparedExecution{
		Task:           task,
		Execution:      execution,
		ExecutionRaw:   executionRaw,
		RuntimeName:    runtimeName,
		RuntimeVersion: runtimeVersion,
	}, nil
}
