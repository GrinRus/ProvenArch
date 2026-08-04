package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/qa"
	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const (
	qaAnswerFile      = "qa-answer.json"
	qaContextPackFile = "context-pack.json"
)

func (s *Service) runQAWithID(
	ctx context.Context,
	request RunRequest,
	runID string,
	initialInfo RunInfo,
	initialArtifacts []Artifact,
	resolvedStepProviders acpruntime.StepProviderResolution,
	resolvedProviderModels acpruntime.ProviderModelResolution,
) (RunInfo, []Artifact, error) {
	startedAt := initialInfo.StartedAt
	if startedAt.IsZero() {
		startedAt = s.clock().UTC()
	}
	stepID := acpruntime.StepIDQAAsk
	question := strings.TrimSpace(request.Question)
	if question == "" {
		return s.failRunBeforeExecution(
			runID,
			initialInfo,
			initialArtifacts,
			fmt.Errorf("question is required for qa runs"),
			fmt.Errorf("question is required for qa runs"),
			"qa run failed: question validation",
			nil,
		)
	}

	qaRoot := runtimeQATaskRoot(runID)
	contextPackRel := path.Join(qaRoot, qaContextPackFile)
	contextPackAbs, err := request.Workspace.Resolve(contextPackRel)
	if err != nil {
		return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, err, err, "qa run failed: context path", nil)
	}
	contextPack, err := qa.NewService().BuildContextPack(ctx, request.Workspace, question, runID, s.clock().UTC())
	if err != nil {
		return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, err, err, "qa run failed: context pack", nil)
	}
	contextRaw, err := json.MarshalIndent(contextPack, "", "  ")
	if err != nil {
		return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, err, err, "qa run failed: context pack marshal", nil)
	}
	contextRaw = append(contextRaw, '\n')
	if err := request.Workspace.WriteFile(contextPackRel, contextRaw); err != nil {
		return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, err, err, "qa run failed: context pack write", nil)
	}

	resolvedExecution := s.ResolveExecutionProfile(request.Workspace.Manifest)
	resolvedPermissions := acpruntime.ResolvePermissions(request.Workspace.Manifest)
	resolvedTimeouts := acpruntime.ResolveTimeouts(request.Workspace.Manifest)
	stepRunnerResolver := newStepRunnerResolver(s.runnerFactory, resolvedStepProviders.Effective)

	artifacts := append([]Artifact(nil), initialArtifacts...)
	artifacts = append(artifacts, Artifact{Path: contextPackRel, Kind: "qa-context", Label: "QA Context Pack"})
	execution := pipelineExecution{
		runID:     runID,
		pipeline:  PipelineQA,
		workspace: request.Workspace,
		store:     model.NewStore(request.Workspace),
		compiler:  reports.NewCompiler(request.Workspace),
		clock:     s.clock,
		pipelineRunProgressState: pipelineRunProgressState{
			startedAt: startedAt,
			stepStatus: RunInfo{
				RunID:                runID,
				Pipeline:             string(PipelineQA),
				Status:               RunStatusRunning,
				StartedAt:            startedAt,
				Question:             question,
				CurrentStep:          stepID,
				StepProviders:        resolvedStepProviders.Effective.StringMap(),
				ProviderModels:       resolvedProviderModels.Effective,
				ProviderModelSources: resolvedProviderModels.Source,
			},
			warnings: []string{},
		},
		pipelineArtifactRegistry: pipelineArtifactRegistry{
			artifacts:     artifacts,
			artifactIndex: artifactIndexFor(artifacts),
		},
		pipelineRuntimeState: pipelineRuntimeState{
			runnerResolver:    stepRunnerResolver,
			runtimeVersions:   map[string]struct{}{},
			resolvedRepoPaths: map[string]string{},
			stepProviders:     resolvedStepProviders.Effective,
			providerModels:    resolvedProviderModels.Effective,
			permissionProfile: resolvedPermissions.Effective,
			executionProfile:  resolvedExecution.Effective,
		},
		pipelineQualityState: pipelineQualityState{
			runtimeStepMetrics:      []runtimeStepQuality{},
			runtimeRecoveryCounters: runtimeRecoveryCounters{},
		},
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			domainRuns:    map[string]domainRunSummary{},
			reportContext: reports.DefaultReportRenderContext(),
		},
	}
	if resolvedTimeouts.Effective.StepTimeoutSec > 0 {
		execution.runtimeStepTimeout = time.Duration(resolvedTimeouts.Effective.StepTimeoutSec) * time.Second
	}
	if resolvedTimeouts.Effective.HeartbeatSec > 0 {
		execution.runtimeHeartbeatInterval = time.Duration(resolvedTimeouts.Effective.HeartbeatSec) * time.Second
	}
	execution.onLog = func(entry RunLogEntry) {
		if strings.TrimSpace(entry.StepID) == "" {
			entry.StepID = stepID
		}
		s.appendRunLog(runID, entry)
	}
	execution.onPermissions = func(pending []acpruntime.PermissionRequest) {
		progress := initialInfo
		progress.Status = RunStatusRunning
		progress.Question = question
		progress.CurrentStep = stepID
		progress.StepProviders = execution.stepProviders.StringMap()
		progress.Warnings = append([]string(nil), execution.warnings...)
		progress.PendingPermissions = append([]acpruntime.PermissionRequest(nil), pending...)
		if err := s.storeRun(runRecord{
			info:      progress,
			artifacts: append([]Artifact(nil), execution.artifacts...),
		}); err != nil {
			execution.recordProgressPersistenceError(err)
		}
	}

	progress := initialInfo
	progress.Status = RunStatusRunning
	progress.Question = question
	progress.CurrentStep = stepID
	progress.StepProviders = execution.stepProviders.StringMap()
	if err := s.storeRun(runRecord{
		info:      progress,
		artifacts: append([]Artifact(nil), execution.artifacts...),
	}); err != nil {
		return initialInfo, initialArtifacts, fmt.Errorf("persist qa progress: %w", err)
	}
	execution.logInfo(stepID, "", "qa context pack prepared", map[string]any{
		"context_pack_path": contextPackRel,
		"context_pack_abs":  contextPackAbs,
		"documents":         len(contextPack.Documents),
		"step_providers":    execution.stepProviders.StringMap(),
	})

	prepared, err := defaultRuntimeTaskExecutor{execution: &execution}.RunRuntimeTask(ctx, RuntimeTaskRequest{
		StepID:          stepID,
		TaskSuffix:      "ask",
		PathScopes:      []string{qaContextPackFile},
		Question:        question,
		ContextPackPath: contextPackAbs,
	})
	if err != nil {
		return s.finishQAFailure(runID, initialInfo, &execution, err)
	}
	if err := execution.progressPersistenceError(); err != nil {
		return s.finishQAFailure(runID, initialInfo, &execution, fmt.Errorf("persist qa permission state: %w", err))
	}
	executionPath := runtimeExecutionMetadataPathForTask(prepared.Task)
	if err := execution.persistRuntimeExecutionArtifact(executionPath, stepID+".runtime-execution", prepared.ExecutionRaw); err != nil {
		return s.finishQAFailure(runID, initialInfo, &execution, err)
	}
	runtimeExecution, err := execution.applyRuntimeTaskExecution(stepID, "", prepared)
	if err != nil {
		return s.finishQAFailure(runID, initialInfo, &execution, err)
	}
	answerRel := path.Join(runtimeExecution.Task.ArtifactRoot, qaAnswerFile)
	answerRaw, readErr := request.Workspace.ReadFile(answerRel)
	if readErr != nil {
		return s.finishQAFailure(runID, initialInfo, &execution, qaRuntimeContractError(runtimeExecution.RuntimeName, readErr))
	}
	answer, parseErr := qa.ParseAnswer(answerRaw)
	if parseErr != nil {
		return s.finishQAFailure(runID, initialInfo, &execution, qaRuntimeContractError(runtimeExecution.RuntimeName, parseErr))
	}
	if validationErr := qa.ValidateAnswerAgainstContext(answer, contextPack); validationErr != nil {
		return s.finishQAFailure(runID, initialInfo, &execution, qaRuntimeContractError(runtimeExecution.RuntimeName, validationErr))
	}
	execution.addArtifacts(Artifact{Path: answerRel, Kind: "qa-answer", Label: "QA Answer"})
	execution.logInfo(stepID, "", "qa run answered", map[string]any{
		"provider":        answer.Provider,
		"runtime_name":    runtimeExecution.RuntimeName,
		"runtime_version": runtimeExecution.RuntimeVersion,
		"citations":       len(answer.Citations),
		"unresolved":      len(answer.Unresolved),
		"confidence":      answer.Confidence,
	})
	return s.finishQASuccess(runID, initialInfo, &execution)
}

func qaRuntimeContractError(runtimeName string, err error) error {
	if err == nil {
		return nil
	}
	provider := acpruntime.Provider(strings.TrimSpace(runtimeName))
	if strings.TrimSpace(string(provider)) == "" {
		provider = acpruntime.ProviderClaudeCode
	}
	return acpruntime.WrapRunnerError(provider, acpruntime.ErrorCodeRuntimeContract, err.Error(), err)
}

func (s *Service) finishQAFailure(runID string, initialInfo RunInfo, execution *pipelineExecution, err error) (RunInfo, []Artifact, error) {
	finishedAt := s.clock().UTC()
	failedInfo := initialInfo
	failedInfo.CurrentStep = acpruntime.StepIDQAAsk
	failedInfo.Question = strings.TrimSpace(initialInfo.Question)
	failedInfo.StepProviders = execution.stepProviders.StringMap()
	failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, err)
	failedInfo.Status = terminalStatusForErrorCode(failedInfo.ErrorCode)
	failedInfo.Warnings = append([]string(nil), execution.warnings...)
	failedInfo.PendingPermissions = append([]acpruntime.PermissionRequest(nil), execution.pendingPermissions...)
	failedInfo.FinishedAt = &finishedAt
	if persistErr := s.storeRun(runRecord{
		info:      failedInfo,
		artifacts: append([]Artifact(nil), execution.artifacts...),
	}); persistErr != nil {
		return failedInfo, execution.artifacts, errors.Join(err, fmt.Errorf("persist failed qa run state: %w", persistErr))
	}
	execution.logError(acpruntime.StepIDQAAsk, "", "qa run failed", map[string]any{
		"error_code": failedInfo.ErrorCode,
		"error":      failedInfo.Error,
	})
	_ = s.cleanupRunLogs()
	return failedInfo, execution.artifacts, err
}

func (s *Service) finishQASuccess(runID string, initialInfo RunInfo, execution *pipelineExecution) (RunInfo, []Artifact, error) {
	finishedAt := s.clock().UTC()
	succeeded := initialInfo
	succeeded.Status = RunStatusSucceeded
	succeeded.CurrentStep = acpruntime.StepIDQAAsk
	succeeded.Question = strings.TrimSpace(initialInfo.Question)
	succeeded.StepProviders = execution.stepProviders.StringMap()
	succeeded.Warnings = append([]string(nil), execution.warnings...)
	succeeded.PendingPermissions = append([]acpruntime.PermissionRequest(nil), execution.pendingPermissions...)
	succeeded.FinishedAt = &finishedAt
	if persistErr := s.storeRun(runRecord{
		info:      succeeded,
		artifacts: append([]Artifact(nil), execution.artifacts...),
	}); persistErr != nil {
		return succeeded, execution.artifacts, fmt.Errorf("persist succeeded qa run state: %w", persistErr)
	}
	execution.logInfo(acpruntime.StepIDQAAsk, "", "qa run succeeded", map[string]any{
		"artifacts": len(execution.artifacts),
	})
	_ = s.cleanupRunLogs()
	return succeeded, execution.artifacts, nil
}
