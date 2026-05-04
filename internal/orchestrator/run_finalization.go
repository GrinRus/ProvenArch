package orchestrator

import (
	"errors"
	"fmt"
	"strings"
)

func (s *Service) failRunBeforeExecution(
	runID string,
	initialInfo RunInfo,
	artifacts []Artifact,
	classifyErr error,
	returnErr error,
	logMessage string,
	warnings []string,
) (RunInfo, []Artifact, error) {
	finishedAt := s.clock().UTC()
	failedInfo := initialInfo
	failedInfo.Status = RunStatusFailed
	failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, classifyErr)
	if warnings != nil {
		failedInfo.Warnings = append([]string(nil), warnings...)
	}
	failedInfo.FinishedAt = &finishedAt
	s.storeRun(runRecord{
		info:      failedInfo,
		artifacts: append([]Artifact(nil), artifacts...),
	})
	fields := map[string]any{
		"error_code": failedInfo.ErrorCode,
		"error":      failedInfo.Error,
	}
	if len(failedInfo.Warnings) > 0 {
		fields["warnings"] = failedInfo.Warnings
	}
	s.appendRunLog(runID, RunLogEntry{
		Timestamp: finishedAt,
		Level:     RunLogLevelError,
		Message:   logMessage,
		Fields:    fields,
	})
	_ = s.cleanupRunLogs()
	return failedInfo, nil, returnErr
}

func (s *Service) finishExecutionFailure(runID string, initialInfo RunInfo, execution *pipelineExecution, err error) (RunInfo, []Artifact, error) {
	finishedAt := s.clock().UTC()
	failedInfo := initialInfo
	failedInfo.Status = RunStatusFailed
	failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, err)
	failedInfo.CurrentStep = execution.stepStatus.CurrentStep
	execution.rewriteTerminalReports(RunStatusFailed)
	failedInfo.Warnings = append([]string(nil), execution.warnings...)
	failedInfo.FinishedAt = &finishedAt
	if qualityArtifact, qualityErr := execution.writeRunQualitySummary(
		RunStatusFailed,
		failedInfo.ErrorCode,
		failedInfo.Error,
		classifyRunFailureSummary(execution.stepStatus.CurrentStep, err),
	); qualityErr == nil {
		execution.addArtifacts(qualityArtifact)
	} else {
		failedInfo.Warnings = append(failedInfo.Warnings, fmt.Sprintf("run quality summary write failed: %v", qualityErr))
		execution.logWarn(execution.stepStatus.CurrentStep, "", "failed to write run quality summary", map[string]any{
			"error": qualityErr.Error(),
		})
	}
	s.storeRun(runRecord{
		info:      failedInfo,
		artifacts: execution.artifacts,
	})
	execution.logError(execution.stepStatus.CurrentStep, "", "run failed", map[string]any{
		"error_code": failedInfo.ErrorCode,
		"error":      failedInfo.Error,
	})
	_ = s.cleanupRunLogs()
	return failedInfo, execution.artifacts, err
}

func (s *Service) finishPartialExecutionFailure(runID string, initialInfo RunInfo, execution *pipelineExecution) (RunInfo, []Artifact, error) {
	finishedAt := s.clock().UTC()
	failedInfo := initialInfo
	failedInfo.Status = RunStatusFailed
	failedInfo.ErrorCode = runErrorCodePartialFailed
	failedInfo.Error = summarizePartialFailures(execution.partialFailures)
	failedInfo.CurrentStep = execution.stepStatus.CurrentStep
	execution.rewriteTerminalReports(RunStatusFailed)
	failedInfo.Warnings = append([]string(nil), execution.warnings...)
	failedInfo.FinishedAt = &finishedAt
	if qualityArtifact, qualityErr := execution.writeRunQualitySummary(
		RunStatusFailed,
		failedInfo.ErrorCode,
		failedInfo.Error,
		runFailureClassification{
			Class:      failedInfo.ErrorCode,
			StepID:     strings.TrimSpace(execution.stepStatus.CurrentStep),
			ShortCause: strings.TrimSpace(failedInfo.Error),
			Source:     "orchestrator.partial_failures",
		},
	); qualityErr == nil {
		execution.addArtifacts(qualityArtifact)
	} else {
		failedInfo.Warnings = append(failedInfo.Warnings, fmt.Sprintf("run quality summary write failed: %v", qualityErr))
	}
	s.storeRun(runRecord{
		info:      failedInfo,
		artifacts: execution.artifacts,
	})
	execution.logError(execution.stepStatus.CurrentStep, "", "run failed: partial shard failures detected", map[string]any{
		"error_code":            failedInfo.ErrorCode,
		"partial_failure_count": len(execution.partialFailures),
	})
	_ = s.cleanupRunLogs()
	return failedInfo, execution.artifacts, errors.New(failedInfo.Error)
}

func (s *Service) finishExecutionSuccess(runID string, initialInfo RunInfo, execution *pipelineExecution) (RunInfo, []Artifact, error) {
	finishedAt := s.clock().UTC()
	succeeded := initialInfo
	succeeded.Status = RunStatusSucceeded
	succeeded.CurrentStep = execution.stepStatus.CurrentStep
	succeeded.Warnings = append([]string(nil), execution.warnings...)
	succeeded.FinishedAt = &finishedAt
	if qualityArtifact, qualityErr := execution.writeRunQualitySummary(RunStatusSucceeded, "", "", runFailureClassification{}); qualityErr == nil {
		execution.addArtifacts(qualityArtifact)
	} else {
		succeeded.Warnings = append(succeeded.Warnings, fmt.Sprintf("run quality summary write failed: %v", qualityErr))
		execution.logWarn(execution.stepStatus.CurrentStep, "", "failed to write run quality summary", map[string]any{
			"error": qualityErr.Error(),
		})
	}
	s.storeRun(runRecord{
		info:      succeeded,
		artifacts: execution.artifacts,
	})
	execution.logInfo(execution.stepStatus.CurrentStep, "", "run succeeded", map[string]any{
		"artifacts": len(execution.artifacts),
	})
	_ = s.cleanupRunLogs()
	return succeeded, execution.artifacts, nil
}

func (e *pipelineExecution) rewriteTerminalReports(status RunStatus) {
	renderCtx := e.terminalRenderContext(status)
	if !renderCtx.IsIncomplete() {
		return
	}

	reportStep := strings.TrimSpace(e.stepStatus.CurrentStep)
	if reportStep == "" {
		reportStep = string(e.pipeline) + ".terminal"
	}

	logRewriteWarning := func(stage string, err error) {
		e.addWarning(fmt.Sprintf("terminal report rewrite failed (%s): %v", stage, err))
		e.logWarn(reportStep, "", "terminal report rewrite failed", map[string]any{
			"stage": stage,
			"error": err.Error(),
		})
	}

	if artifacts, err := e.compiler.WriteCoverage(e.coverage, e.questions, renderCtx); err != nil {
		logRewriteWarning("coverage", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}

	if artifacts, err := e.compiler.WriteFindings(e.findings, renderCtx); err != nil {
		logRewriteWarning("findings", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}

	if domainReports, err := e.authoredDomainReports(); err != nil {
		logRewriteWarning("domain-outputs.prepare", err)
	} else if artifacts, err := e.compiler.WriteDomainOutputs(domainReports); err != nil {
		logRewriteWarning("domain-outputs", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}

	if artifacts, err := e.compiler.WriteDomainTaskEnvelopes(e.stagedDomainEnvelopes()); err != nil {
		logRewriteWarning("domain-task-envelopes", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}
}
