package orchestrator

import (
	"strings"

	"github.com/GrinRus/ProvenArch/internal/reports"
)

type runtimeShardOutcome struct {
	PlannedShards   int
	SucceededShards int
	FailedShards    int
}

func (o runtimeShardOutcome) merge(other runtimeShardOutcome) runtimeShardOutcome {
	return runtimeShardOutcome{
		PlannedShards:   o.PlannedShards + other.PlannedShards,
		SucceededShards: o.SucceededShards + other.SucceededShards,
		FailedShards:    o.FailedShards + other.FailedShards,
	}
}

func (o runtimeShardOutcome) phaseState() reports.EvidencePhaseState {
	state := reports.EvidencePhaseState{
		PlannedShards:   o.PlannedShards,
		SucceededShards: o.SucceededShards,
		FailedShards:    o.FailedShards,
		Status:          reports.EvidenceStatusUsable,
	}
	switch {
	case o.PlannedShards > 0 && o.SucceededShards == 0:
		state.Status = reports.EvidenceStatusUnusable
	case o.FailedShards > 0:
		state.Status = reports.EvidenceStatusPartial
	default:
		state.Status = reports.EvidenceStatusUsable
	}
	return state
}

func (e *pipelineExecution) recordRuntimeStepOutcome(stepID string, outcome runtimeShardOutcome) {
	switch {
	case strings.HasSuffix(stepID, "step1.collect"):
		e.collectOutcome = e.collectOutcome.merge(outcome)
	case strings.HasSuffix(stepID, "step3.findings"):
		e.findingsOutcome = e.findingsOutcome.merge(outcome)
	}
	e.refreshReportContext()
}

func (e *pipelineExecution) markFindingsSkipped(reason string) {
	e.findingsSkipped = true
	e.addReportReason(reason)
	e.refreshReportContext()
}

func (e *pipelineExecution) addReportReason(reason string) {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return
	}
	for _, existing := range e.reportContext.Reasons {
		if existing == trimmed {
			return
		}
	}
	e.reportContext.Reasons = append(e.reportContext.Reasons, trimmed)
}

func (e *pipelineExecution) refreshReportContext() {
	ctx := reports.DefaultReportRenderContext()
	ctx.Collect = e.collectOutcome.phaseState()
	if e.findingsSkipped {
		ctx.Findings = reports.EvidencePhaseState{
			Status: reports.EvidenceStatusSkipped,
		}
	} else {
		ctx.Findings = e.findingsOutcome.phaseState()
	}

	reasons := make([]string, 0, 4)
	switch ctx.Collect.Status {
	case reports.EvidenceStatusPartial:
		reasons = append(reasons, "collect_partial_shard_failures")
	case reports.EvidenceStatusUnusable:
		reasons = append(reasons, "collect_all_shards_failed")
	}
	if e.findingsSkipped {
		reasons = append(reasons, "findings_skipped_due_to_unusable_collect")
	} else {
		switch ctx.Findings.Status {
		case reports.EvidenceStatusPartial:
			reasons = append(reasons, "findings_partial_shard_failures")
		case reports.EvidenceStatusUnusable:
			reasons = append(reasons, "findings_all_shards_failed")
		}
	}
	ctx.Reasons = append(reasons, e.reportContext.Reasons...)
	if ctx.Collect.Status != reports.EvidenceStatusUsable || ctx.Findings.Status != reports.EvidenceStatusUsable {
		ctx.ReportMode = reports.ReportModeIncomplete
	} else {
		ctx.ReportMode = reports.ReportModeNormal
	}
	e.reportContext = reports.NormalizeReportRenderContext(ctx)
}

func (e *pipelineExecution) renderContext() reports.ReportRenderContext {
	return reports.NormalizeReportRenderContext(e.reportContext)
}

func (e *pipelineExecution) terminalRenderContext(status RunStatus) reports.ReportRenderContext {
	ctx := e.renderContext()
	if e.shouldMarkFindingsSkippedAtTerminal(status) {
		ctx.Findings = reports.EvidencePhaseState{
			Status: reports.EvidenceStatusSkipped,
		}
		ctx.Reasons = append(ctx.Reasons, "findings_not_executed_due_to_pipeline_abort")
		ctx.ReportMode = reports.ReportModeIncomplete
	}
	return reports.NormalizeReportRenderContext(ctx)
}

func (e *pipelineExecution) shouldSkipFindingsRuntime() bool {
	return e.renderContext().Collect.Status == reports.EvidenceStatusUnusable
}

func (e *pipelineExecution) shouldMarkFindingsSkippedAtTerminal(status RunStatus) bool {
	if status != RunStatusFailed {
		return false
	}
	if e.findingsSkipped || e.findingsOutcome.PlannedShards > 0 {
		return false
	}
	currentIndex := pipelineStepIndex(e.pipeline, e.stepStatus.CurrentStep)
	findingsIndex := pipelineStepIndex(e.pipeline, string(e.pipeline)+".step3.findings")
	if findingsIndex < 0 {
		return false
	}
	return currentIndex < 0 || currentIndex <= findingsIndex
}

func pipelineStepIndex(pipeline Pipeline, stepID string) int {
	stepID = strings.TrimSpace(stepID)
	if stepID == "" {
		return -1
	}
	for idx, candidate := range stepIDsForPipeline(pipeline) {
		if candidate == stepID {
			return idx
		}
	}
	return -1
}
