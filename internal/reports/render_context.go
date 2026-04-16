package reports

type EvidenceStatus string

const (
	EvidenceStatusUsable   EvidenceStatus = "usable"
	EvidenceStatusPartial  EvidenceStatus = "partial"
	EvidenceStatusUnusable EvidenceStatus = "unusable"
	EvidenceStatusSkipped  EvidenceStatus = "skipped"
)

type ReportMode string

const (
	ReportModeNormal     ReportMode = "normal"
	ReportModeIncomplete ReportMode = "incomplete"
)

type EvidencePhaseState struct {
	Status          EvidenceStatus `json:"status"`
	PlannedShards   int            `json:"planned_shards"`
	SucceededShards int            `json:"succeeded_shards"`
	FailedShards    int            `json:"failed_shards"`
}

type ReportRenderContext struct {
	Collect    EvidencePhaseState `json:"collect"`
	Findings   EvidencePhaseState `json:"findings"`
	ReportMode ReportMode         `json:"report_mode"`
	Reasons    []string           `json:"reasons,omitempty"`
}

func DefaultReportRenderContext() ReportRenderContext {
	return ReportRenderContext{
		Collect: EvidencePhaseState{
			Status: EvidenceStatusUsable,
		},
		Findings: EvidencePhaseState{
			Status: EvidenceStatusUsable,
		},
		ReportMode: ReportModeNormal,
	}
}

func NormalizeReportRenderContext(ctx ReportRenderContext) ReportRenderContext {
	ctx.Collect = normalizeEvidencePhaseState(ctx.Collect, false)
	ctx.Findings = normalizeEvidencePhaseState(ctx.Findings, true)
	if ctx.ReportMode == "" {
		if ctx.Collect.Status != EvidenceStatusUsable || ctx.Findings.Status != EvidenceStatusUsable {
			ctx.ReportMode = ReportModeIncomplete
		} else {
			ctx.ReportMode = ReportModeNormal
		}
	}
	ctx.Reasons = uniqueSorted(ctx.Reasons)
	return ctx
}

func (ctx ReportRenderContext) IsIncomplete() bool {
	return NormalizeReportRenderContext(ctx).ReportMode == ReportModeIncomplete
}

func normalizeEvidencePhaseState(state EvidencePhaseState, allowSkipped bool) EvidencePhaseState {
	switch state.Status {
	case EvidenceStatusUsable, EvidenceStatusPartial, EvidenceStatusUnusable:
	case EvidenceStatusSkipped:
		if !allowSkipped {
			state.Status = EvidenceStatusUsable
		}
	default:
		state.Status = EvidenceStatusUsable
	}
	if state.PlannedShards < 0 {
		state.PlannedShards = 0
	}
	if state.SucceededShards < 0 {
		state.SucceededShards = 0
	}
	if state.FailedShards < 0 {
		state.FailedShards = 0
	}
	return state
}
